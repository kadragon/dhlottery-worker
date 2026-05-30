package dhlottery

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/notify"
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
	checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{})

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

func TestCheckWinningNotifiesWins(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: ledgerFixture(t)})
	col := &notify.Collector{}
	results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), col)
	if len(results) != 3 {
		t.Fatalf("expected 3 wins, got %d", len(results))
	}

	payloads := col.Payloads()
	if len(payloads) != 3 {
		t.Fatalf("expected 3 payloads, got %d", len(payloads))
	}
	for _, p := range payloads {
		if p.Type != notify.Success || p.Title != "복권 당첨!" {
			t.Errorf("payload = %+v", p)
		}
	}
	// Jackpot row (로또 1등 2,000,000,000) is the third win.
	jackpot := payloads[2]
	if !strings.Contains(jackpot.Message, "1223") {
		t.Errorf("message = %q", jackpot.Message)
	}
	if detailValue(jackpot, "prizeAmount") != "2000000000" {
		t.Errorf("prizeAmount = %q", detailValue(jackpot, "prizeAmount"))
	}
	if detailValue(jackpot, "product") != "로또6/45" {
		t.Errorf("product = %q", detailValue(jackpot, "product"))
	}
}

func TestCheckWinningNoWin(t *testing.T) {
	body := `{"data":{"total":1,"list":[{"ltGdsCd":"LO40","ltGdsNm":"로또6/45","ltEpsd":1225,"ltWnResult":"낙첨","ltWnAmt":0,"wnRnk":null}]}}`
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: body})
	col := &notify.Collector{}
	results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), col)
	if len(results) != 0 {
		t.Errorf("expected no wins, got %+v", results)
	}
	if !col.IsEmpty() {
		t.Error("should not notify when there is no win")
	}
}

func TestCheckWinningEmptyList(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: `{"data":{"total":0,"list":[]}}`})
	col := &notify.Collector{}
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), col); len(results) != 0 {
		t.Errorf("expected empty, got %+v", results)
	}
	if !col.IsEmpty() {
		t.Error("should not notify on empty list")
	}
}

func TestCheckWinningFetchFailure(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 500, Body: "error"})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{}); len(results) != 0 {
		t.Errorf("expected empty on fetch failure, got %+v", results)
	}
}

func TestCheckWinningParseFailure(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: "<html>not json</html>"})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{}); len(results) != 0 {
		t.Errorf("expected empty on parse failure, got %+v", results)
	}
}

func TestCheckWinningRedirect(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{
		Status: 302,
		Header: http.Header{"Location": {"https://www.dhlottery.co.kr/errorPage"}},
		Body:   "",
	})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{}); len(results) != 0 {
		t.Errorf("expected empty on redirect, got %+v", results)
	}
}
