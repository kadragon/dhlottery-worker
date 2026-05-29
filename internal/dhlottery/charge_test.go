package dhlottery

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/notify"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func balanceJSON(amount int) string {
	return fmt.Sprintf(`{"data":{"userMndp":{"crntEntrsAmt":%d}}}`, amount)
}

// chargeClient drives the round+balance fetches followed by the kbank fetch.
func chargeClient(balance int, kbankStatus int) (*httpclient.Client, *testutil.StubDoer) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(
		testutil.JSON(roundJSON),
		testutil.JSON(balanceJSON(balance)),
		testutil.StubResponse{Status: kbankStatus, Body: "<html></html>"},
	)}
	return httpclient.NewWithDoer(stub), stub
}

func kbankRequests(stub *testutil.StubDoer) []testutil.RecordedRequest {
	var out []testutil.RecordedRequest
	for _, r := range stub.Requests {
		if strings.Contains(r.URL, "kbank.do") {
			out = append(out, r)
		}
	}
	return out
}

func TestCheckDepositSufficient(t *testing.T) {
	for _, balance := range []int{constants.MinDepositAmount, 50000} {
		client, stub := chargeClient(balance, 200)
		col := &notify.Collector{}
		ok, err := checkDeposit(client, constants.MinDepositAmount, col)
		if err != nil || !ok {
			t.Fatalf("balance %d: ok=%v err=%v", balance, ok, err)
		}
		if len(kbankRequests(stub)) != 0 {
			t.Errorf("balance %d: should not initialize charge", balance)
		}
		if !col.IsEmpty() {
			t.Errorf("balance %d: should not notify", balance)
		}
	}
}

func TestCheckDepositInsufficientWarns(t *testing.T) {
	client, stub := chargeClient(4000, 200)
	col := &notify.Collector{}
	ok, err := checkDeposit(client, constants.MinDepositAmount, col)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	kb := kbankRequests(stub)
	if len(kb) != 1 {
		t.Fatalf("expected exactly 1 charge GET, got %d", len(kb))
	}
	if kb[0].Method != http.MethodGet {
		t.Errorf("charge request method = %s, want GET", kb[0].Method)
	}
	if kb[0].URL != chargeInitURL {
		t.Errorf("charge URL = %s, want %s", kb[0].URL, chargeInitURL)
	}
	if !strings.Contains(kb[0].URL, "GoodsAmt=50000") {
		t.Errorf("charge URL should request 50000: %s", kb[0].URL)
	}
	if ua := kb[0].Header.Get("User-Agent"); ua != constants.UserAgent {
		t.Errorf("charge UA = %q", ua)
	}

	payloads := col.Payloads()
	if len(payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(payloads))
	}
	p := payloads[0]
	if p.Type != notify.Warning || p.Title != "예치금 부족" {
		t.Errorf("payload = %+v", p)
	}
	if !strings.Contains(p.Message, "입금") {
		t.Errorf("message should request deposit: %q", p.Message)
	}
	if detailValue(p, "currentBalance") != "4,000원" {
		t.Errorf("currentBalance = %q", detailValue(p, "currentBalance"))
	}
	if detailValue(p, "minimumRequired") != "5,000원" {
		t.Errorf("minimumRequired = %q", detailValue(p, "minimumRequired"))
	}
}

func TestCheckDepositZeroBalance(t *testing.T) {
	client, _ := chargeClient(0, 200)
	ok, err := checkDeposit(client, constants.MinDepositAmount, &notify.Collector{})
	if err != nil || ok {
		t.Errorf("ok=%v err=%v", ok, err)
	}
}

func TestCheckDepositChargeInitFailure(t *testing.T) {
	client, _ := chargeClient(3000, 500)
	col := &notify.Collector{}
	ok, err := checkDeposit(client, constants.MinDepositAmount, col)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	payloads := col.Payloads()
	if len(payloads) != 1 || payloads[0].Type != notify.Error || payloads[0].Title != "충전 초기화 실패" {
		t.Errorf("expected error payload, got %+v", payloads)
	}
}

func TestCheckDepositCustomThreshold(t *testing.T) {
	client, _ := chargeClient(constants.WeeklyCombinedRequiredBalance, 200)
	col := &notify.Collector{}
	ok, err := checkDeposit(client, constants.WeeklyCombinedRequiredBalance, col)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !col.IsEmpty() {
		t.Error("should not notify when balance meets custom threshold")
	}
}

func TestCheckDepositCustomThresholdWarn(t *testing.T) {
	client, _ := chargeClient(9000, 200)
	col := &notify.Collector{}
	checkDeposit(client, constants.WeeklyCombinedRequiredBalance, col)
	p := col.Payloads()[0]
	if detailValue(p, "minimumRequired") != "10,000원" {
		t.Errorf("minimumRequired = %q, want 10,000원", detailValue(p, "minimumRequired"))
	}
}

func TestCheckDepositAccountFetchError(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.StubResponse{Status: 500, Body: "err"}))
	col := &notify.Collector{}
	_, err := checkDeposit(client, constants.MinDepositAmount, col)
	if err == nil {
		t.Error("expected error to propagate (fail-safe)")
	}
	if !col.IsEmpty() {
		t.Error("should not notify when account fetch fails")
	}
}

func detailValue(p notify.Payload, key string) string {
	for _, kv := range p.Details {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}
