package dhlottery

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/notify"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
	"golang.org/x/text/encoding/korean"
)

func winningFixtureUTF8(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "winning-results.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func toEUCKR(t *testing.T, utf8 string) string {
	t.Helper()
	encoded, err := korean.EUCKR.NewEncoder().Bytes([]byte(utf8))
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func parseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestParseWinningResults(t *testing.T) {
	results := parseWinningResultsFromHTML(winningFixtureUTF8(t))
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	if results[0].RoundNumber != 1144 || results[0].Rank != 1 || results[0].PrizeAmount != 2000000000 {
		t.Errorf("results[0] = %+v", results[0])
	}
	if results[0].MatchCount == nil || *results[0].MatchCount != 6 {
		t.Errorf("results[0].MatchCount = %v", results[0].MatchCount)
	}
	if results[1].RoundNumber != 1144 || results[1].Rank != 2 || results[1].PrizeAmount != 65000000 {
		t.Errorf("results[1] = %+v", results[1])
	}
	if results[1].MatchCount == nil || *results[1].MatchCount != 5 {
		t.Errorf("results[1].MatchCount = %v", results[1].MatchCount)
	}
}

func TestParseWinningSkipsInvalidRows(t *testing.T) {
	// No detailPop -> round number unparseable -> row skipped.
	html := `<tr>
		<td>2025-12-10</td><td>로또6/45</td><td><a href="#">상세</a></td>
		<td>5</td><td>1등 (일치 6개)</td><td>1,000,000,000원</td><td>2025-12-13</td>
	</tr>`
	if results := parseWinningResultsFromHTML(html); len(results) != 0 {
		t.Errorf("expected no results, got %+v", results)
	}
}

func TestParseWinningMultiRow(t *testing.T) {
	html := `
		<tr><td>2025-12-10</td><td>로또6/45</td>
		<td><a href="javascript:detailPop('x','y','1140');">상세</a></td>
		<td>5</td><td>1등 (일치 6개)</td><td>1,000,000,000원</td><td>2025-12-13</td></tr>
		<tr><td>2025-12-11</td><td>로또6/45</td>
		<td><a href="javascript:detailPop('x','y','1141');">상세</a></td>
		<td>3</td><td>2등 (일치 5개)</td><td>50,000,000원</td><td>2025-12-14</td></tr>
		<tr><td>2025-12-12</td><td>로또6/45</td>
		<td><a href="javascript:detailPop('x','y','1142');">상세</a></td>
		<td>2</td><td>3등 (일치 5개)</td><td>1,500,000원</td><td>2025-12-15</td></tr>`
	results := parseWinningResultsFromHTML(html)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].RoundNumber != 1140 || results[1].RoundNumber != 1141 || results[2].RoundNumber != 1142 {
		t.Errorf("rounds = %d,%d,%d", results[0].RoundNumber, results[1].RoundNumber, results[2].RoundNumber)
	}
}

func TestFilterJackpotWins(t *testing.T) {
	results := parseWinningResultsFromHTML(winningFixtureUTF8(t))
	jackpot := filterJackpotWins(results)
	if len(jackpot) != 1 || jackpot[0].Rank != 1 {
		t.Errorf("filterJackpotWins = %+v", jackpot)
	}
	if got := filterJackpotWins(nil); len(got) != 0 {
		t.Errorf("filterJackpotWins(nil) = %+v", got)
	}
}

func checkClient(resp testutil.StubResponse) (*httpclient.Client, *testutil.StubDoer) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(resp)}
	return httpclient.NewWithDoer(stub), stub
}

func TestCheckWinningURLParams(t *testing.T) {
	client, stub := checkClient(testutil.StubResponse{Status: 200, Body: toEUCKR(t, winningFixtureUTF8(t))})
	checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{})

	if len(stub.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(stub.Requests))
	}
	u := stub.Requests[0].URL
	for _, want := range []string{"myPage.do", "method=lottoBuyList", "searchStartDate=", "searchEndDate="} {
		if !strings.Contains(u, want) {
			t.Errorf("URL missing %q: %s", want, u)
		}
	}
	if stub.Requests[0].Method != http.MethodGet {
		t.Errorf("method = %s, want GET", stub.Requests[0].Method)
	}
}

func TestCheckWinningNotifiesJackpot(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: toEUCKR(t, winningFixtureUTF8(t))})
	col := &notify.Collector{}
	results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), col)
	if len(results) != 1 {
		t.Fatalf("expected 1 jackpot, got %d", len(results))
	}
	p := col.Payloads()[0]
	if p.Type != notify.Success || p.Title != "로또 당첨!" {
		t.Errorf("payload = %+v", p)
	}
	if !strings.Contains(p.Message, "1144") {
		t.Errorf("message = %q", p.Message)
	}
	if detailValue(p, "prizeAmount") != "2000000000" {
		t.Errorf("prizeAmount = %q", detailValue(p, "prizeAmount"))
	}
}

func TestCheckWinningNoWin(t *testing.T) {
	noWin := strings.Replace(strings.Replace(winningFixtureUTF8(t), "1등", "2등", 1), "2등", "3등", 1)
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: toEUCKR(t, noWin)})
	col := &notify.Collector{}
	results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), col)
	if len(results) != 0 {
		t.Errorf("expected no wins, got %+v", results)
	}
	if !col.IsEmpty() {
		t.Error("should not notify when no rank-1 win")
	}
}

func TestCheckWinningFetchFailure(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 500, Body: "error"})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{}); len(results) != 0 {
		t.Errorf("expected empty on fetch failure, got %+v", results)
	}
}

func TestCheckWinningParseFailure(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{Status: 200, Body: "<html>no rows</html>"})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{}); len(results) != 0 {
		t.Errorf("expected empty on parse failure, got %+v", results)
	}
}

func TestCheckWinningRedirect(t *testing.T) {
	client, _ := checkClient(testutil.StubResponse{
		Status: 302,
		Header: http.Header{"Location": {"https://www.dhlottery.co.kr/login"}},
		Body:   toEUCKR(t, winningFixtureUTF8(t)),
	})
	if results := checkWinning(client, parseTime(t, "2025-12-15T10:00:00+09:00"), &notify.Collector{}); len(results) != 0 {
		t.Errorf("expected empty on redirect, got %+v", results)
	}
}
