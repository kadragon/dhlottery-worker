package dhlottery

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/datekst"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
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
	client, _ := checkClient(testutil.StubResponse{Status: 500, Body: "error"})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on fetch failure, got (%+v, %v)", s, ok)
	}
}

func TestAggregateLedgerRedirect(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{
		Status: 302,
		Header: http.Header{"Location": {"https://www.dhlottery.co.kr/errorPage"}},
	})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on redirect, got (%+v, %v)", s, ok)
	}
}

func TestAggregateLedgerParseFailure(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: "<html>not json</html>"})
	if s, ok := aggregateLedger(client, aggRecentStart, aggNow(t)); ok || s != (LedgerSummary{}) {
		t.Errorf("expected (zero, false) on parse failure, got (%+v, %v)", s, ok)
	}
}

// A failure on a later page discards the whole aggregation (all-or-nothing):
// page 1 succeeds, page 2 returns 500, so the result is (zero, false).
func TestAggregateLedgerMidPageFailure(t *testing.T) {
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
	if len(stub.Requests) != 2 {
		t.Errorf("expected 2 requests before bailing, got %d", len(stub.Requests))
	}
}
