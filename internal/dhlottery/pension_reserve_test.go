package dhlottery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/dherr"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/notify"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

const pensionSession = "12345678901234567890123456789012.tail.of.session=="

// encResp builds an EL response whose q field is the real-encrypted canned JSON
// (the production code decrypts it with the matching session).
func encResp(t *testing.T, jsonStr string) testutil.StubResponse {
	t.Helper()
	q, err := EncryptElQ(jsonStr, pensionSession)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"q": q})
	return testutil.StubResponse{Status: 200, Body: string(body)}
}

// recordDecrypted decrypts an incoming EL request body and stores the parsed
// plaintext form so the test can assert on it.
func recordDecrypted(t *testing.T, r testutil.RecordedRequest, key string, into map[string]url.Values) {
	t.Helper()
	body, err := url.ParseQuery(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := DecryptElQ(body.Get("q"), pensionSession)
	if err != nil {
		t.Fatalf("decrypt request q: %v", err)
	}
	form, err := url.ParseQuery(plain)
	if err != nil {
		t.Fatal(err)
	}
	into[key] = form
}

type pensionScenario struct {
	deposit  string
	dup      string
	add      string
	firstErr error
}

func pensionClient(t *testing.T, s pensionScenario) (*httpclient.Client, map[string]url.Values, *testutil.StubDoer) {
	decrypted := map[string]url.Values{}
	stub := &testutil.StubDoer{Handler: func(_ int, r testutil.RecordedRequest) (testutil.StubResponse, error) {
		switch {
		case strings.Contains(r.URL, "TotalGame.jsp"):
			if s.firstErr != nil {
				return testutil.StubResponse{}, s.firstErr
			}
			return testutil.JSON("{}"), nil
		case strings.Contains(r.URL, "reserveGame.jsp"):
			return testutil.JSON("{}"), nil
		case strings.Contains(r.URL, "roundRemainTime.do"):
			return testutil.JSON(`{"resultCode":"100","resultMsg":"ok","ROUND":"303","DRAW_DATE":"2026-02-19"}`), nil
		case strings.Contains(r.URL, "checkDeposit.do"):
			recordDecrypted(t, r, "checkDeposit", decrypted)
			return encResp(t, s.deposit), nil
		case strings.Contains(r.URL, "checkMyReserve.do"):
			recordDecrypted(t, r, "checkMyReserve", decrypted)
			return encResp(t, s.dup), nil
		case strings.Contains(r.URL, "addMyReserve.do"):
			recordDecrypted(t, r, "addMyReserve", decrypted)
			return encResp(t, s.add), nil
		}
		return testutil.StubResponse{}, fmt.Errorf("unexpected %s", r.URL)
	}}
	client := httpclient.NewWithDoer(stub)
	client.SetCookie("JSESSIONID", pensionSession)
	return client, decrypted, stub
}

func TestPensionReserveSuccess(t *testing.T) {
	client, decrypted, _ := pensionClient(t, pensionScenario{
		deposit: `{"resultCode":"100","resultMsg":"조회 성공","deposit":"10000"}`,
		dup:     `{"resultCode":"100","resultMsg":"예약 조회 성공","doubleRound":[]}`,
		add:     `{"resultCode":"100","resultMsg":"예약 성공","reserveOrderNo":"A-1","reserveOrderDate":"2026-02-18 17:00:00"}`,
	})
	col := &notify.Collector{}
	out := reservePensionNextWeek(client, col)

	if out.Status != "success" || !out.Success || out.TargetRound != 304 || out.TotalAmount != 5000 {
		t.Fatalf("outcome = %+v", out)
	}

	// Real-crypto round trip: the server-side decrypt of our request must show
	// the correct next round and win date.
	if decrypted["checkMyReserve"].Get("nextRound") != "304" ||
		decrypted["checkMyReserve"].Get("repeatRoundCnt") != "1" ||
		decrypted["checkMyReserve"].Get("winDate") != "" {
		t.Errorf("checkMyReserve form = %v", decrypted["checkMyReserve"])
	}
	if decrypted["addMyReserve"].Get("winDate") != "2026.02.26" {
		t.Errorf("addMyReserve winDate = %q, want 2026.02.26", decrypted["addMyReserve"].Get("winDate"))
	}

	p := col.Payloads()[0]
	if p.Type != notify.Success || p.Title != "연금복권 예약 완료" || detailValue(p, "예약번호") != "A-1" {
		t.Errorf("payload = %+v", p)
	}
}

func TestPensionReserveDuplicate(t *testing.T) {
	client, _, stub := pensionClient(t, pensionScenario{
		deposit: `{"resultCode":"100","deposit":"10000"}`,
		dup:     `{"resultCode":"100","doubleRound":[{"doubleRound":"304","doubleCnt":"5"}]}`,
	})
	col := &notify.Collector{}
	out := reservePensionNextWeek(client, col)

	if out.Status != "skipped" || !out.Success || !out.Skipped || out.TargetRound != 304 {
		t.Fatalf("outcome = %+v", out)
	}
	if _, called := findRequest(stub, "addMyReserve"); called {
		t.Error("addMyReserve should not be called when duplicate exists")
	}
	if p := col.Payloads()[0]; p.Type != notify.Warning || p.Title != "연금복권 예약 건너뜀" {
		t.Errorf("payload = %+v", p)
	}
}

func TestPensionReserveInsufficientDeposit(t *testing.T) {
	client, _, _ := pensionClient(t, pensionScenario{
		deposit: `{"resultCode":"100","deposit":"3000"}`,
	})
	col := &notify.Collector{}
	out := reservePensionNextWeek(client, col)

	if out.Status != "failure" || out.Code != "PENSION_INSUFFICIENT_DEPOSIT" || out.TargetRound != 304 {
		t.Fatalf("outcome = %+v", out)
	}
	if p := col.Payloads()[0]; p.Type != notify.Warning || p.Title != "연금복권 예약 건너뜀" {
		t.Errorf("payload = %+v", p)
	}
}

func TestPensionReserveDuplicateAuthError(t *testing.T) {
	client, _, _ := pensionClient(t, pensionScenario{
		deposit: `{"resultCode":"100","deposit":"10000"}`,
		dup:     `{"resultCode":"E001","resultMsg":"로그인후 이용하시기 바랍니다.","doubleRound":[]}`,
	})
	col := &notify.Collector{}
	out := reservePensionNextWeek(client, col)

	if out.Status != "failure" || out.Code != "E001" || out.TargetRound != 304 {
		t.Fatalf("outcome = %+v", out)
	}
	if p := col.Payloads()[0]; p.Type != notify.Error || p.Title != "연금복권 예약 실패" {
		t.Errorf("payload = %+v", p)
	}
}

func TestPensionReserveBootstrapError(t *testing.T) {
	client, _, _ := pensionClient(t, pensionScenario{
		firstErr: dherr.New("EL session failed", "PENSION_BOOTSTRAP_FAILED"),
	})
	col := &notify.Collector{}
	out := reservePensionNextWeek(client, col)

	if out.Status != "failure" || out.Code != "PENSION_BOOTSTRAP_FAILED" {
		t.Fatalf("outcome = %+v", out)
	}
	if p := col.Payloads()[0]; p.Type != notify.Error || detailValue(p, "오류코드") != "PENSION_BOOTSTRAP_FAILED" {
		t.Errorf("payload = %+v", p)
	}
}

func TestPensionReserveGenericError(t *testing.T) {
	client, _, _ := pensionClient(t, pensionScenario{
		firstErr: errors.New("fetch failed"),
	})
	col := &notify.Collector{}
	out := reservePensionNextWeek(client, col)

	if out.Status != "failure" || out.Code != "PENSION_UNEXPECTED_ERROR" {
		t.Fatalf("outcome = %+v", out)
	}
	if p := col.Payloads()[0]; p.Type != notify.Error || detailValue(p, "오류코드") != "PENSION_UNEXPECTED_ERROR" {
		t.Errorf("payload = %+v", p)
	}
}

func TestPensionReserveMissingSession(t *testing.T) {
	// No JSESSIONID/DHJSESSIONID cookie -> postEncrypted fails at getSessionID.
	stub := &testutil.StubDoer{Handler: func(_ int, r testutil.RecordedRequest) (testutil.StubResponse, error) {
		switch {
		case strings.Contains(r.URL, "TotalGame.jsp"), strings.Contains(r.URL, "reserveGame.jsp"):
			return testutil.JSON("{}"), nil
		case strings.Contains(r.URL, "roundRemainTime.do"):
			return testutil.JSON(`{"resultCode":"100","ROUND":"303","DRAW_DATE":"2026-02-19"}`), nil
		}
		return testutil.StubResponse{}, fmt.Errorf("unexpected %s", r.URL)
	}}
	client := httpclient.NewWithDoer(stub)
	col := &notify.Collector{}
	out := reservePensionNextWeek(client, col)
	if out.Status != "failure" || out.Code != "PENSION_AUTH_MISSING" {
		t.Errorf("outcome = %+v", out)
	}
}
