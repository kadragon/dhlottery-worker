package dhlottery

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/datekst"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func ledgerFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "selectMyLotteryledger-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func parseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func checkClient(resp testutil.StubResponse) (*httpclient.Client, *testutil.StubDoer) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(resp)}
	return httpclient.NewWithDoer(stub), stub
}

// extractWins keeps only rows with a positive win amount (ltWnAmt > 0). Lost
// rows (ltWnAmt == 0) and undrawn rows (ltWnAmt == null) are ignored.
func TestExtractWins(t *testing.T) {
	var data ledgerResponse
	if err := json.Unmarshal([]byte(ledgerFixture(t)), &data); err != nil {
		t.Fatal(err)
	}
	wins := extractWins(data.Data.List)
	if len(wins) != 3 {
		t.Fatalf("expected 3 wins, got %d: %+v", len(wins), wins)
	}

	// rowId 3: 로또 5등 5,000원
	if wins[0].RoundNumber != 1224 || wins[0].Rank != 5 || wins[0].PrizeAmount != 5000 || wins[0].Product != "로또6/45" {
		t.Errorf("wins[0] = %+v", wins[0])
	}
	// rowId 5: 연금 2등 1,000,000원
	if wins[1].RoundNumber != 316 || wins[1].Rank != 2 || wins[1].PrizeAmount != 1000000 || wins[1].Product != "연금복권720+" {
		t.Errorf("wins[1] = %+v", wins[1])
	}
	// rowId 6: 로또 1등 2,000,000,000원
	if wins[2].RoundNumber != 1223 || wins[2].Rank != 1 || wins[2].PrizeAmount != 2000000000 || wins[2].Product != "로또6/45" {
		t.Errorf("wins[2] = %+v", wins[2])
	}
}

func TestExtractWinsEmpty(t *testing.T) {
	if got := extractWins(nil); len(got) != 0 {
		t.Errorf("extractWins(nil) = %+v", got)
	}
	// Only lost / undrawn rows -> no wins.
	rows := []ledgerRow{
		{LtEpsd: 100, LtWnResult: "낙첨", LtWnAmt: intPtr(0)},
		{LtEpsd: 101, LtWnResult: "미추첨", LtWnAmt: nil},
	}
	if got := extractWins(rows); len(got) != 0 {
		t.Errorf("expected no wins, got %+v", got)
	}
}

func intPtr(n int) *int { return &n }

func TestCheckWinningURLParams(t *testing.T) {
	client, stub := checkClient(testutil.StubResponse{Status: 200, Body: ledgerFixture(t)})
	checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"))

	if len(stub.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(stub.Requests))
	}
	u := stub.Requests[0].URL
	// Previous week of Mon 2025-12-15 KST is 2025-12-08 .. 2025-12-14.
	for _, want := range []string{
		"selectMyLotteryledger.do",
		"srchStrDt=20251208",
		"srchEndDt=20251214",
		"pageNum=1",
		"recordCountPerPage=50",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("URL missing %q: %s", want, u)
		}
	}
	if stub.Requests[0].Method != http.MethodGet {
		t.Errorf("method = %s, want GET", stub.Requests[0].Method)
	}
}

func TestCheckWinningReturnsWins(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: ledgerFixture(t)})
	results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"))
	if len(results) != 3 {
		t.Fatalf("expected 3 wins, got %d", len(results))
	}
}

func TestCheckWinningNoWin(t *testing.T) {
	body := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","ltGdsNm":"로또6/45","ltEpsd":1225,"ltWnResult":"낙첨","ltWnAmt":0,"wnRnk":null}]}}`
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: body})
	results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"))
	if len(results) != 0 {
		t.Errorf("expected no wins, got %+v", results)
	}
}

func TestCheckWinningEmptyList(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: `{"data":{"total":0,"list":[]}}`})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00")); len(results) != 0 {
		t.Errorf("expected empty, got %+v", results)
	}
}

func TestCheckWinningFetchFailure(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 500, Body: "error"})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00")); len(results) != 0 {
		t.Errorf("expected empty on fetch failure, got %+v", results)
	}
}

func TestCheckWinningParseFailure(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: "<html>not json</html>"})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00")); len(results) != 0 {
		t.Errorf("expected empty on parse failure, got %+v", results)
	}
}

func TestCheckWinningRedirect(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{
		Status: 302,
		Header: http.Header{"Location": {"https://www.dhlottery.co.kr/errorPage"}},
		Body:   "",
	})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00")); len(results) != 0 {
		t.Errorf("expected empty on redirect, got %+v", results)
	}
}

// aggNow is a fixed "now"; aggRecentStart sits inside one ledgerWindowDays
// window of it, so single-window tests make exactly one request.
const aggRecentStart = "20260401"

func aggNow(t *testing.T) time.Time { return parseTime(t, "2026-06-08T10:00:00+09:00") }

// noLedgerSleep stubs the ledger retry backoff and returns the recorded delays.
func noLedgerSleep(t *testing.T) *[]time.Duration {
	t.Helper()
	var slept []time.Duration
	orig := ledgerSleep
	ledgerSleep = func(d time.Duration) { slept = append(slept, d) }
	t.Cleanup(func() { ledgerSleep = orig })
	return &slept
}

func TestAggregateLedgerFixture(t *testing.T) {
	client, stub := checkClient(testutil.StubResponse{Status: 200, Body: ledgerFixture(t)})
	s, ok := aggregateLedger(client, aggRecentStart, aggNow(t))
	if !ok {
		t.Fatal("expected ok=true on a successful fetch")
	}

	// 6 rows: prchsQty 5+5+1+1+1+1 = 14 units × 1000 = 14,000.
	if s.CumulativePurchase != 14000 {
		t.Errorf("CumulativePurchase = %d, want 14000", s.CumulativePurchase)
	}
	// wins: 5,000 + 1,000,000 + 2,000,000,000 = 2,001,005,000.
	if s.CumulativeWinning != 2001005000 {
		t.Errorf("CumulativeWinning = %d, want 2001005000", s.CumulativeWinning)
	}
	if len(stub.Requests) != 1 {
		t.Fatalf("expected 1 request (single window, one page), got %d", len(stub.Requests))
	}
	u := stub.Requests[0].URL
	for _, want := range []string{"srchStrDt=" + aggRecentStart, "srchEndDt=20260608", "pageNum=1", "recordCountPerPage=100"} {
		if !strings.Contains(u, want) {
			t.Errorf("URL missing %q: %s", want, u)
		}
	}
}

func TestAggregateLedgerPaging(t *testing.T) {
	page1 := `{"data":{"total":3,"list":[{"ltGdsCd":"LO40","prchsQty":5,"ltWnAmt":null},{"ltGdsCd":"LO40","prchsQty":1,"ltWnAmt":5000}]}}`
	page2 := `{"data":{"total":3,"list":[{"ltGdsCd":"LP72","prchsQty":1,"ltWnAmt":1000000}]}}`
	stub := &testutil.StubDoer{Handler: testutil.Sequence(
		testutil.StubResponse{Status: 200, Body: page1},
		testutil.StubResponse{Status: 200, Body: page2},
	)}
	client := httpclient.NewWithDoer(stub)

	s, ok := aggregateLedger(client, aggRecentStart, aggNow(t))
	if !ok {
		t.Fatal("expected ok=true")
	}

	if len(stub.Requests) != 2 {
		t.Fatalf("expected 2 requests (one window, two pages), got %d", len(stub.Requests))
	}
	if !strings.Contains(stub.Requests[0].URL, "pageNum=1") || !strings.Contains(stub.Requests[1].URL, "pageNum=2") {
		t.Errorf("page sequence = %q, %q", stub.Requests[0].URL, stub.Requests[1].URL)
	}
	if s.CumulativePurchase != 7000 { // (5+1+1)×1000
		t.Errorf("CumulativePurchase = %d, want 7000", s.CumulativePurchase)
	}
	if s.CumulativeWinning != 1005000 { // 5000 + 1,000,000
		t.Errorf("CumulativeWinning = %d, want 1005000", s.CumulativeWinning)
	}
}

// A span longer than ledgerWindowDays is walked in contiguous, non-overlapping
// windows (newest first), ending today and bottoming out at startDate. Each
// window here returns one 1000-purchase row, so the total equals the window
// count.
func TestAggregateLedgerChunksContiguous(t *testing.T) {
	body := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":1,"ltWnAmt":0}]}}`
	stub := &testutil.StubDoer{Handler: testutil.Sequence(testutil.StubResponse{Status: 200, Body: body})}
	client := httpclient.NewWithDoer(stub)

	const start = "20251201" // > ledgerWindowDays before now → multiple windows
	s, ok := aggregateLedger(client, start, aggNow(t))
	if !ok {
		t.Fatal("expected ok=true")
	}

	n := len(stub.Requests)
	if n < 2 {
		t.Fatalf("expected ≥2 windows for a >90d span, got %d", n)
	}
	if s.CumulativePurchase != n*1000 {
		t.Errorf("CumulativePurchase = %d, want %d (one row per window)", s.CumulativePurchase, n*1000)
	}

	windows := make([][2]string, n)
	for i, req := range stub.Requests {
		u, err := url.Parse(req.URL)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		windows[i] = [2]string{q.Get("srchStrDt"), q.Get("srchEndDt")}
	}
	if windows[0][1] != "20260608" {
		t.Errorf("first window end = %s, want 20260608 (today)", windows[0][1])
	}
	for i := 1; i < n; i++ {
		wantEnd := strings.ReplaceAll(datekst.AddDaysToYmd(windows[i-1][0], -1), "-", "")
		if windows[i][1] != wantEnd {
			t.Errorf("window %d end = %s, want %s (contiguous, no overlap)", i, windows[i][1], wantEnd)
		}
	}
	if last := windows[n-1][0]; last != start {
		t.Errorf("last window start = %s, want %s (clamped to startDate)", last, start)
	}
}

func TestAggregateLedgerEmpty(t *testing.T) {
	client, stub := checkClient(testutil.StubResponse{Status: 200, Body: `{"data":{"total":0,"list":[]}}`})
	s, ok := aggregateLedger(client, aggRecentStart, aggNow(t))
	if !ok {
		t.Fatal("expected ok=true on an empty-but-successful fetch")
	}
	if s.CumulativePurchase != 0 || s.CumulativeWinning != 0 {
		t.Errorf("summary = %+v, want zero", s)
	}
	if len(stub.Requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(stub.Requests))
	}
}

func TestAggregateLedgerStartAfterNow(t *testing.T) {
	client, stub := checkClient(testutil.StubResponse{Status: 200, Body: ledgerFixture(t)})
	s, ok := aggregateLedger(client, "20991231", aggNow(t))
	if !ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, true) when start > now, got (%+v, %v)", s, ok)
	}
	if len(stub.Requests) != 0 {
		t.Errorf("expected no requests when start > now, got %d", len(stub.Requests))
	}
}

func TestAggregateLedgerFetchFailure(t *testing.T) {
	noLedgerSleep(t)
	client, _ := checkClient(testutil.StubResponse{Status: 500, Body: "error"})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on fetch failure, got (%+v, %v)", s, ok)
	}
}

func TestAggregateLedgerRedirect(t *testing.T) {
	noLedgerSleep(t)
	client, _ := checkClient(testutil.StubResponse{
		Status: 302,
		Header: http.Header{"Location": {"https://www.dhlottery.co.kr/errorPage"}},
	})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on redirect, got (%+v, %v)", s, ok)
	}
}

func TestAggregateLedgerParseFailure(t *testing.T) {
	noLedgerSleep(t)
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: "<html>not json</html>"})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on parse failure, got (%+v, %v)", s, ok)
	}
}

// A failure on a later page discards the whole aggregation (all-or-nothing):
// page 1 succeeds, page 2 returns 500 on both the attempt and its retry, so the
// result is (zero, false).
func TestAggregateLedgerMidPageFailure(t *testing.T) {
	noLedgerSleep(t)
	page1 := `{"data":{"total":3,"list":[{"ltGdsCd":"LO40","prchsQty":5,"ltWnAmt":5000},{"ltGdsCd":"LO40","prchsQty":1,"ltWnAmt":null}]}}`
	stub := &testutil.StubDoer{Handler: testutil.Sequence(
		testutil.StubResponse{Status: 200, Body: page1},
		testutil.StubResponse{Status: 500, Body: "error"},
	)}
	client := httpclient.NewWithDoer(stub)

	s, ok := aggregateLedger(client, aggRecentStart, aggNow(t))
	if ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) when a later page fails, got (%+v, %v)", s, ok)
	}
	if len(stub.Requests) != 3 {
		t.Errorf("expected 3 requests (page 1, page 2 + retry) before bailing, got %d", len(stub.Requests))
	}
}

// All-or-nothing across windows: window 1 succeeds, window 2's fetch fails
// (attempt and retry), so the whole walk returns (zero, false) — the partial
// window-1 sum is discarded.
func TestAggregateLedgerMidWindowFailure(t *testing.T) {
	noLedgerSleep(t)
	win1 := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":5,"ltWnAmt":5000}]}}`
	stub := &testutil.StubDoer{Handler: testutil.Sequence(
		testutil.StubResponse{Status: 200, Body: win1},    // window 1, page 1
		testutil.StubResponse{Status: 500, Body: "error"}, // window 2, page 1 → fail
	)}
	client := httpclient.NewWithDoer(stub)

	s, ok := aggregateLedger(client, "20251201", aggNow(t)) // >90d span → ≥2 windows
	if ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) when a later window fails, got (%+v, %v)", s, ok)
	}
	if len(stub.Requests) != 3 {
		t.Errorf("expected 3 requests (window 1 ok, window 2 fails + retry), got %d", len(stub.Requests))
	}
}

// Pins ledgerWindowDays: a span of exactly that many days fits one window; one
// more day forces a second. A regression here (e.g. window size widened past the
// API's silent cap) is the bug this PR fixes.
func TestAggregateLedgerWindowSizeBoundary(t *testing.T) {
	body := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":1,"ltWnAmt":0}]}}`
	now := aggNow(t) // 2026-06-08
	end := "2026-06-08"
	cstr := func(ymd string) string { return strings.ReplaceAll(ymd, "-", "") }

	for _, tc := range []struct {
		name     string
		start    string
		wantReqs int
	}{
		{"exactly one window", cstr(datekst.AddDaysToYmd(end, -(ledgerWindowDays - 1))), 1},
		{"just over one window", cstr(datekst.AddDaysToYmd(end, -ledgerWindowDays)), 2},
	} {
		stub := &testutil.StubDoer{Handler: testutil.Sequence(testutil.StubResponse{Status: 200, Body: body})}
		client := httpclient.NewWithDoer(stub)
		if _, ok := aggregateLedger(client, tc.start, now); !ok {
			t.Fatalf("%s: ok=false", tc.name)
		}
		if len(stub.Requests) != tc.wantReqs {
			t.Errorf("%s (start=%s): requests = %d, want %d", tc.name, tc.start, len(stub.Requests), tc.wantReqs)
		}
	}
}

// A malformed start date must report failure, not a zero summary tagged as real.
func TestAggregateLedgerInvalidStart(t *testing.T) {
	client, stub := checkClient(testutil.StubResponse{Status: 200, Body: ledgerFixture(t)})
	for _, bad := range []string{"foo", "2026-6-1", "", "20261332"} {
		if s, ok := aggregateLedger(client, bad, aggNow(t)); ok || s != (LedgerSummary{}) {
			t.Errorf("start %q: expected (zero, false), got (%+v, %v)", bad, s, ok)
		}
	}
	if len(stub.Requests) != 0 {
		t.Errorf("expected no requests for invalid start, got %d", len(stub.Requests))
	}
}

// A start older than the maxWindows backstop reports failure rather than a
// silently-partial total.
func TestAggregateLedgerBackstopExhausted(t *testing.T) {
	body := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":1,"ltWnAmt":0}]}}`
	stub := &testutil.StubDoer{Handler: testutil.Sequence(testutil.StubResponse{Status: 200, Body: body})}
	client := httpclient.NewWithDoer(stub)

	if s, ok := aggregateLedger(client, "19000101", aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on backstop exhaustion, got (%+v, %v)", s, ok)
	}
}

// A single transient page failure is absorbed by one retry after a backoff;
// the aggregation still completes with the full total.
func TestAggregateLedgerTransientFailureRetried(t *testing.T) {
	slept := noLedgerSleep(t)
	body := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":5,"ltWnAmt":5000}]}}`
	stub := &testutil.StubDoer{Handler: testutil.Sequence(
		testutil.StubResponse{Status: 500, Body: "error"},
		testutil.StubResponse{Status: 200, Body: body},
	)}
	client := httpclient.NewWithDoer(stub)

	s, ok := aggregateLedger(client, aggRecentStart, aggNow(t))
	if !ok {
		t.Fatal("expected ok=true after a successful retry")
	}
	if s.CumulativePurchase != 5000 || s.CumulativeWinning != 5000 {
		t.Errorf("summary = %+v, want purchase 5000 / winning 5000", s)
	}
	if len(stub.Requests) != 2 {
		t.Errorf("expected 2 requests (fail + retry), got %d", len(stub.Requests))
	}
	if stub.Requests[0].URL != stub.Requests[1].URL {
		t.Errorf("retry URL differs: %q vs %q", stub.Requests[0].URL, stub.Requests[1].URL)
	}
	if len(*slept) != 1 || (*slept)[0] != ledgerRetryDelay {
		t.Errorf("slept = %v, want one %v backoff", *slept, ledgerRetryDelay)
	}
}

// Two consecutive failures on the same page exhaust the single retry and fail
// the whole aggregation (all-or-nothing preserved).
func TestAggregateLedgerRetryExhausted(t *testing.T) {
	noLedgerSleep(t)
	client, stub := checkClient(testutil.StubResponse{Status: 500, Body: "error"})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) after retry exhausted, got (%+v, %v)", s, ok)
	}
	if len(stub.Requests) != 2 {
		t.Errorf("expected exactly 2 requests (attempt + one retry), got %d", len(stub.Requests))
	}
}

// captureLogs routes warn/error logs into a buffer for the test's duration.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	logger.SetWriters(&buf, &buf)
	t.Cleanup(func() { logger.SetWriters(os.Stdout, os.Stderr) })
	return &buf
}

// logEvents returns the level of each logged line keyed by its event field.
func logEvents(t *testing.T, buf *bytes.Buffer) map[string]string {
	t.Helper()
	events := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var entry struct {
			Level string `json:"level"`
			Event string `json:"event"`
		}
		if line == "" || json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}
		events[entry.Event] = entry.Level
	}
	return events
}

// A retried first failure is a warning, not an error: only the final failure
// is reported at error level.
func TestAggregateLedgerRetryAttemptLoggedAsWarn(t *testing.T) {
	noLedgerSleep(t)
	logs := captureLogs(t)
	body := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":5}]}}`
	stub := &testutil.StubDoer{Handler: testutil.Sequence(
		testutil.StubResponse{Status: 503, Body: "busy"},
		testutil.StubResponse{Status: 200, Body: body},
	)}
	if _, ok := aggregateLedger(httpclient.NewWithDoer(stub), aggRecentStart, aggNow(t)); !ok {
		t.Fatal("expected ok=true after a successful retry")
	}
	events := logEvents(t, logs)
	if events["ledger_retry_attempt"] != "warn" {
		t.Errorf("events = %v, want ledger_retry_attempt at warn", events)
	}
	for ev, level := range events {
		if level == "error" {
			t.Errorf("unexpected error-level log %q after a recovered retry", ev)
		}
	}
}

// A transient failure that also fails on retry warns once, then reports the
// final failure at error level.
func TestAggregateLedgerRetryFailureLogLevels(t *testing.T) {
	noLedgerSleep(t)
	logs := captureLogs(t)
	client, _ := checkClient(testutil.StubResponse{Status: 503, Body: "busy"})
	if _, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok {
		t.Fatal("expected ok=false after retry exhausted")
	}
	events := logEvents(t, logs)
	if events["ledger_retry_attempt"] != "warn" || events["ledger_aggregate_fetch_failed"] != "error" {
		t.Errorf("events = %v, want ledger_retry_attempt=warn and ledger_aggregate_fetch_failed=error", events)
	}
	if !strings.Contains(logs.String(), `"srchStrDt"`) {
		t.Errorf("failure logs lack the window: %s", logs.String())
	}
}

// Transport errors are transient: retried once.
func TestAggregateLedgerTransportErrorRetried(t *testing.T) {
	noLedgerSleep(t)
	stub := &testutil.StubDoer{Handler: func(n int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
		if n == 0 {
			return testutil.StubResponse{}, errors.New("connection reset")
		}
		return testutil.StubResponse{Status: 200, Body: `{"data":{"total":0,"list":[]}}`}, nil
	}}
	if _, ok := aggregateLedger(httpclient.NewWithDoer(stub), aggRecentStart, aggNow(t)); !ok {
		t.Fatal("expected ok=true after retrying a transport error")
	}
	if len(stub.Requests) != 2 {
		t.Errorf("expected 2 requests (error + retry), got %d", len(stub.Requests))
	}
}

// Permanent failures (non-transient status, redirect, unparseable body) fail
// immediately without a wasted retry.
func TestAggregateLedgerPermanentFailureNotRetried(t *testing.T) {
	cases := map[string]testutil.StubResponse{
		"404":      {Status: 404, Body: "not found"},
		"redirect": {Status: 302, Header: http.Header{"Location": {"/login"}}},
		"bad json": {Status: 200, Body: "<html>maintenance</html>"},
	}
	for name, resp := range cases {
		t.Run(name, func(t *testing.T) {
			slept := noLedgerSleep(t)
			logs := captureLogs(t)
			client, stub := checkClient(resp)
			if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
				t.Errorf("expected (zero, false), got (%+v, %v)", s, ok)
			}
			if len(stub.Requests) != 1 || len(*slept) != 0 {
				t.Errorf("requests=%d slept=%v, want 1 request and no backoff", len(stub.Requests), *slept)
			}
			if _, retried := logEvents(t, logs)["ledger_retry_attempt"]; retried {
				t.Error("permanent failure logged a retry attempt")
			}
		})
	}
}

// Responses whose data.total contradicts the rows read so far fail the window
// (realtest confirmed data.total tracks the row count, so these mean a short or
// garbled read) and log ledger_total_anomaly at error level.
func TestAggregateLedgerTotalAnomalyFails(t *testing.T) {
	cases := map[string]func(int, testutil.RecordedRequest) (testutil.StubResponse, error){
		"total zero with rows": testutil.Sequence(
			testutil.StubResponse{Status: 200, Body: `{"data":{"total":0,"list":[{"ltGdsCd":"LO40","prchsQty":5}]}}`},
		),
		"total missing with rows": testutil.Sequence(
			testutil.StubResponse{Status: 200, Body: `{"data":{"list":[{"ltGdsCd":"LO40","prchsQty":5}]}}`},
		),
		"empty page before total": func(n int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
			if n == 0 {
				return testutil.StubResponse{Status: 200, Body: `{"data":{"total":150,"list":[{"ltGdsCd":"LO40","prchsQty":5}]}}`}, nil
			}
			return testutil.StubResponse{Status: 200, Body: `{"data":{"total":150,"list":[]}}`}, nil
		},
		"empty first page with total": testutil.Sequence(
			testutil.StubResponse{Status: 200, Body: `{"data":{"total":3,"list":[]}}`},
		),
	}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			logs := captureLogs(t)
			stub := &testutil.StubDoer{Handler: handler}
			if s, ok := aggregateLedger(httpclient.NewWithDoer(stub), aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
				t.Errorf("expected (zero, false), got (%+v, %v)", s, ok)
			}
			if logEvents(t, logs)["ledger_total_anomaly"] != "error" {
				t.Errorf("missing ledger_total_anomaly error; logs: %s", logs.String())
			}
		})
	}
}

// Over-delivery and a drifting data.total do not lose rows, so the window
// still completes; they only log a ledger_total_anomaly warning.
func TestAggregateLedgerTotalAnomalyWarns(t *testing.T) {
	cases := map[string]func(int, testutil.RecordedRequest) (testutil.StubResponse, error){
		"rows exceed total": testutil.Sequence(
			testutil.StubResponse{Status: 200, Body: `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","prchsQty":3},{"ltGdsCd":"LO40","prchsQty":2}]}}`},
		),
		"total increased between pages": func(n int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
			if n == 0 {
				return testutil.StubResponse{Status: 200, Body: `{"data":{"total":2,"list":[{"ltGdsCd":"LO40","prchsQty":3}]}}`}, nil
			}
			return testutil.StubResponse{Status: 200, Body: `{"data":{"total":3,"list":[{"ltGdsCd":"LO40","prchsQty":1},{"ltGdsCd":"LO40","prchsQty":1}]}}`}, nil
		},
		// The empty page matches the latest total, not the stale page-1 one.
		"total decreased then empty page": func(n int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
			if n == 0 {
				return testutil.StubResponse{Status: 200, Body: `{"data":{"total":2,"list":[{"ltGdsCd":"LO40","prchsQty":5}]}}`}, nil
			}
			return testutil.StubResponse{Status: 200, Body: `{"data":{"total":1,"list":[]}}`}, nil
		},
		"total decreased mid-window": func(n int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
			switch n {
			case 0:
				return testutil.StubResponse{Status: 200, Body: `{"data":{"total":3,"list":[{"ltGdsCd":"LO40","prchsQty":2}]}}`}, nil
			case 1:
				return testutil.StubResponse{Status: 200, Body: `{"data":{"total":2,"list":[{"ltGdsCd":"LO40","prchsQty":3}]}}`}, nil
			}
			return testutil.StubResponse{Status: 200, Body: `{"data":{"total":2,"list":[]}}`}, nil
		},
	}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			logs := captureLogs(t)
			stub := &testutil.StubDoer{Handler: handler}
			s, ok := aggregateLedger(httpclient.NewWithDoer(stub), aggRecentStart, aggNow(t))
			if !ok || s.CumulativePurchase != 5000 {
				t.Errorf("got (%+v, %v), want purchase 5000 and ok=true", s, ok)
			}
			if logEvents(t, logs)["ledger_total_anomaly"] != "warn" {
				t.Errorf("missing ledger_total_anomaly warn; logs: %s", logs.String())
			}
		})
	}
}

// A consistent response logs no anomaly.
func TestAggregateLedgerConsistentTotalNoAnomaly(t *testing.T) {
	logs := captureLogs(t)
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: ledgerFixture(t)})
	if _, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); !ok {
		t.Fatal("expected ok=true")
	}
	if _, found := logEvents(t, logs)["ledger_total_anomaly"]; found {
		t.Errorf("unexpected anomaly log: %s", logs.String())
	}
}

// A server that keeps returning rows past maxPages while data.total claims
// more must not yield a silently-truncated total: the window reports failure.
func TestAggregateLedgerPageBackstopTruncated(t *testing.T) {
	body := `{"data":{"total":1000000,"list":[{"ltGdsCd":"LO40","prchsQty":1,"ltWnAmt":0}]}}`
	client, stub := checkClient(testutil.StubResponse{Status: 200, Body: body})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on page-backstop truncation, got (%+v, %v)", s, ok)
	}
	if len(stub.Requests) != ledgerMaxPages {
		t.Errorf("expected %d requests (page backstop), got %d", ledgerMaxPages, len(stub.Requests))
	}
}
