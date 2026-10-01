package dhlottery

import (
	"net/url"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/checkpoint"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

// With aggNow = 2026-06-08 and ledgerSettleLagDays = 35, the settled cutoff is
// 2026-05-04: rows up to it are folded into the checkpoint; later rows (which
// may still be undrawn) are summed live every run.
const incSettled = "2026-05-04"

// incRow is one ledger page with a single 5-game row winning ltWnAmt.
func incRow(win string) testutil.StubResponse {
	return testutil.JSON(`{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":5,"ltWnAmt":` + win + `}]}}`)
}

func windowsOf(t *testing.T, stub *testutil.StubDoer) [][2]string {
	t.Helper()
	var out [][2]string
	for _, req := range stub.Requests {
		u, err := url.Parse(req.URL)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		out = append(out, [2]string{q.Get("srchStrDt"), q.Get("srchEndDt")})
	}
	return out
}

func TestAggregateLedgerIncrementalFromCheckpoint(t *testing.T) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(incRow("5000"), incRow("null"))}
	client := httpclient.NewWithDoer(stub)
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-04-27", Purchase: 100000, Winning: 5000}

	s, next, ok := aggregateLedgerIncremental(client, "20200101", aggNow(t), prev)
	if !ok {
		t.Fatal("expected ok=true")
	}
	want := [][2]string{{"20260428", "20260504"}, {"20260505", "20260608"}}
	if got := windowsOf(t, stub); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("windows = %v, want %v (only since checkpoint)", got, want)
	}
	if s != (LedgerSummary{CumulativePurchase: 110000, CumulativeWinning: 10000}) {
		t.Errorf("summary = %+v", s)
	}
	wantNext := checkpoint.Checkpoint{Start: "2020-01-01", Through: incSettled, Purchase: 105000, Winning: 10000}
	if next == nil || *next != wantNext {
		t.Errorf("next = %+v, want %+v (unsettled tail excluded)", next, wantNext)
	}
}

func TestAggregateLedgerIncrementalCheckpointAtCutoff(t *testing.T) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(incRow("null"))}
	client := httpclient.NewWithDoer(stub)
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: incSettled, Purchase: 100000, Winning: 5000}

	s, next, ok := aggregateLedgerIncremental(client, "20200101", aggNow(t), prev)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got := windowsOf(t, stub); len(got) != 1 || got[0] != [2]string{"20260505", "20260608"} {
		t.Fatalf("windows = %v, want tail only", got)
	}
	if s != (LedgerSummary{CumulativePurchase: 105000, CumulativeWinning: 5000}) {
		t.Errorf("summary = %+v", s)
	}
	if next == nil || *next != *prev {
		t.Errorf("next = %+v, want unchanged %+v", next, prev)
	}
}

func TestAggregateLedgerIncrementalFallsBackToFullScan(t *testing.T) {
	cases := map[string]*checkpoint.Checkpoint{
		"nil":            nil,
		"start mismatch": {Start: "2020-01-01", Through: "2026-04-27"},
		"future through": {Start: "2026-04-01", Through: "2026-05-10"},
		"through<start":  {Start: "2026-04-01", Through: "2026-03-01"},
		"bad through":    {Start: "2026-04-01", Through: "garbage"},
		"negative total": {Start: "2026-04-01", Through: "2026-04-10", Purchase: -1},
	}
	for name, prev := range cases {
		t.Run(name, func(t *testing.T) {
			stub := &testutil.StubDoer{Handler: testutil.Sequence(incRow("5000"), incRow("null"))}
			client := httpclient.NewWithDoer(stub)

			s, next, ok := aggregateLedgerIncremental(client, "20260401", aggNow(t), prev)
			if !ok {
				t.Fatal("expected ok=true")
			}
			want := [][2]string{{"20260401", "20260504"}, {"20260505", "20260608"}}
			if got := windowsOf(t, stub); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("windows = %v, want full scan %v", got, want)
			}
			if s != (LedgerSummary{CumulativePurchase: 10000, CumulativeWinning: 5000}) {
				t.Errorf("summary = %+v", s)
			}
			wantNext := checkpoint.Checkpoint{Start: "2026-04-01", Through: incSettled, Purchase: 5000, Winning: 5000}
			if next == nil || *next != wantNext {
				t.Errorf("next = %+v, want %+v", next, wantNext)
			}
		})
	}
}

// A checkpoint whose Through carries a misplaced dash still denotes a valid
// date once compacted; it must resume from the day after, not panic.
func TestAggregateLedgerIncrementalOddlyDashedThrough(t *testing.T) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(incRow("0"), incRow("null"))}
	client := httpclient.NewWithDoer(stub)
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026042-7", Purchase: 1000}

	if _, _, ok := aggregateLedgerIncremental(client, "20200101", aggNow(t), prev); !ok {
		t.Fatal("expected ok=true")
	}
	if got := windowsOf(t, stub); len(got) == 0 || got[0][0] != "20260428" {
		t.Errorf("windows = %v, want resume from 20260428", got)
	}
}

// An empty settled delta (no purchases, or a silently empty ledger response)
// must not advance the checkpoint, so the span is re-queried next run.
func TestAggregateLedgerIncrementalEmptyDeltaKeepsThrough(t *testing.T) {
	empty := testutil.JSON(`{"data":{"total":0,"list":[]}}`)
	stub := &testutil.StubDoer{Handler: testutil.Sequence(empty, incRow("null"))}
	client := httpclient.NewWithDoer(stub)
	prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-04-27", Purchase: 1000}

	s, next, ok := aggregateLedgerIncremental(client, "20200101", aggNow(t), prev)
	if !ok || s.CumulativePurchase != 6000 {
		t.Fatalf("summary = %+v ok=%v", s, ok)
	}
	if next == nil || *next != *prev {
		t.Errorf("next = %+v, want unchanged %+v", next, prev)
	}
}

func TestAggregateLedgerIncrementalEmptyFullScanNoCheckpoint(t *testing.T) {
	client, _ := checkClient(testutil.JSON(`{"data":{"total":0,"list":[]}}`))
	if _, next, ok := aggregateLedgerIncremental(client, "20260401", aggNow(t), nil); !ok || next != nil {
		t.Errorf("ok=%v next=%+v, want ok=true, no checkpoint", ok, next)
	}
}

func TestAggregateLedgerIncrementalStartAfterCutoff(t *testing.T) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(incRow("null"))}
	client := httpclient.NewWithDoer(stub)

	s, next, ok := aggregateLedgerIncremental(client, "20260520", aggNow(t), nil)
	if !ok || next != nil {
		t.Fatalf("ok=%v next=%+v, want ok=true, no checkpoint (nothing settled)", ok, next)
	}
	if got := windowsOf(t, stub); len(got) != 1 || got[0] != [2]string{"20260520", "20260608"} {
		t.Errorf("windows = %v", got)
	}
	if s.CumulativePurchase != 5000 {
		t.Errorf("summary = %+v", s)
	}
}

func TestAggregateLedgerIncrementalStartAfterNow(t *testing.T) {
	client, stub := checkClient(incRow("null"))
	s, next, ok := aggregateLedgerIncremental(client, "20991231", aggNow(t), nil)
	if !ok || next != nil || s != (LedgerSummary{}) || len(stub.Requests) != 0 {
		t.Errorf("got (%+v, %+v, %v, %d requests), want zero/nil/true/0", s, next, ok, len(stub.Requests))
	}
}

func TestAggregateLedgerIncrementalInvalidStart(t *testing.T) {
	client, stub := checkClient(incRow("null"))
	if _, next, ok := aggregateLedgerIncremental(client, "foo", aggNow(t), nil); ok || next != nil {
		t.Errorf("ok=%v next=%+v, want failure", ok, next)
	}
	if len(stub.Requests) != 0 {
		t.Errorf("expected no requests, got %d", len(stub.Requests))
	}
}

func TestAggregateLedgerIncrementalFailure(t *testing.T) {
	noLedgerSleep(t)
	for name, failAt := range map[string]int{"settled segment": 0, "tail": 1} {
		t.Run(name, func(t *testing.T) {
			stub := &testutil.StubDoer{Handler: func(call int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
				if call >= failAt {
					return testutil.StubResponse{Status: 500, Body: "error"}, nil
				}
				return incRow("0"), nil
			}}
			client := httpclient.NewWithDoer(stub)
			prev := &checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-04-27"}

			s, next, ok := aggregateLedgerIncremental(client, "20200101", aggNow(t), prev)
			if ok || next != nil || s != (LedgerSummary{}) {
				t.Errorf("got (%+v, %+v, %v), want all-or-nothing failure", s, next, ok)
			}
		})
	}
}

func TestCheckpointResumable(t *testing.T) {
	now := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC) // KST 2026-10-05; settled cutoff 2026-08-31
	cases := map[string]struct {
		cp    *checkpoint.Checkpoint
		start string
		want  bool
	}{
		"valid":               {&checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-08-23"}, "20200101", true},
		"through at cutoff":   {&checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-08-31"}, "20200101", true},
		"through past":        {&checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-09-01"}, "20200101", false},
		"start mismatch":      {&checkpoint.Checkpoint{Start: "2021-01-01", Through: "2026-08-23"}, "20200101", false},
		"nil":                 {nil, "20200101", false},
		"malformed env start": {&checkpoint.Checkpoint{Start: "2020-01-01", Through: "2026-08-23"}, "foo", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := CheckpointResumable(tc.cp, tc.start, now); got != tc.want {
				t.Errorf("CheckpointResumable = %v, want %v", got, tc.want)
			}
		})
	}
}
