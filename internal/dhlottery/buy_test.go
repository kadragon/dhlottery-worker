package dhlottery

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/notify"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

const roundJSON1203 = `{"data":{"result":{"ltEpsd":1203}}}`

var (
	readyOK        = testutil.JSON(`{"direct_yn":"N","ready_ip":"INTCOM2","ready_time":"0","ready_cnt":"0"}`)
	execBuySuccess = testutil.JSON(`{"loginYn":"Y","result":{"resultCode":"100","resultMsg":"Success"}}`)
)

// buyHandler routes the account, ready, and execBuy calls by URL.
func buyHandler(ready, exec testutil.StubResponse, readyErr, execErr error) func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
	return func(_ int, r testutil.RecordedRequest) (testutil.StubResponse, error) {
		switch {
		case strings.Contains(r.URL, "selectThsLt645Info"):
			return testutil.JSON(roundJSON1203), nil
		case strings.Contains(r.URL, "selectUserMndp"):
			return testutil.JSON(balanceJSON(50000)), nil
		case strings.Contains(r.URL, "egovUserReadySocket"):
			if readyErr != nil {
				return testutil.StubResponse{}, readyErr
			}
			return ready, nil
		case strings.Contains(r.URL, "execBuy"):
			if execErr != nil {
				return testutil.StubResponse{}, execErr
			}
			return exec, nil
		}
		return testutil.StubResponse{}, fmt.Errorf("unexpected url %s", r.URL)
	}
}

func buyClient(ready, exec testutil.StubResponse, readyErr, execErr error) (*httpclient.Client, *testutil.StubDoer) {
	stub := &testutil.StubDoer{Handler: buyHandler(ready, exec, readyErr, execErr)}
	return httpclient.NewWithDoer(stub), stub
}

func findRequest(stub *testutil.StubDoer, substr string) (testutil.RecordedRequest, bool) {
	for _, r := range stub.Requests {
		if strings.Contains(r.URL, substr) {
			return r, true
		}
	}
	return testutil.RecordedRequest{}, false
}

func execBuyForm(t *testing.T, stub *testutil.StubDoer) url.Values {
	t.Helper()
	req, ok := findRequest(stub, "execBuy")
	if !ok {
		t.Fatal("execBuy request not found")
	}
	v, err := url.ParseQuery(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestPurchaseReadyFailure(t *testing.T) {
	client, _ := buyClient(testutil.StubResponse{Status: 500}, execBuySuccess, nil, nil)
	out := purchaseLottery(client, &notify.Collector{})
	if out.Success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.Error, "Purchase ready failed") || !strings.Contains(out.Error, "500") {
		t.Errorf("error = %q", out.Error)
	}
}

func TestPurchaseExecutionFailure(t *testing.T) {
	client, _ := buyClient(readyOK, testutil.StubResponse{Status: 502}, nil, nil)
	out := purchaseLottery(client, &notify.Collector{})
	if out.Success || !strings.Contains(out.Error, "Purchase execution failed") || !strings.Contains(out.Error, "502") {
		t.Errorf("out = %+v", out)
	}
}

func TestPurchaseSuccess(t *testing.T) {
	client, stub := buyClient(readyOK, execBuySuccess, nil, nil)
	col := &notify.Collector{}
	out := purchaseLottery(client, col)
	if !out.Success {
		t.Fatalf("expected success, got %+v", out)
	}
	if out.RoundNumber != 1203 || out.GameCount != 5 || out.TotalAmount != 5000 {
		t.Errorf("out = %+v", out)
	}
	if out.PurchaseDate == "" {
		t.Error("PurchaseDate should be set")
	}

	if _, ok := findRequest(stub, "egovUserReadySocket"); !ok {
		t.Error("ready endpoint should be called")
	}

	form := execBuyForm(t, stub)
	if form.Get("direct") != "INTCOM2" {
		t.Errorf("direct = %q", form.Get("direct"))
	}
	if form.Get("gameCnt") != "5" {
		t.Errorf("gameCnt = %q", form.Get("gameCnt"))
	}
	if form.Get("nBuyAmount") != "5000" {
		t.Errorf("nBuyAmount = %q", form.Get("nBuyAmount"))
	}
	if form.Get("saleMdaDcd") != "10" {
		t.Errorf("saleMdaDcd = %q", form.Get("saleMdaDcd"))
	}
	param := form.Get("param")
	if !strings.Contains(param, `"genType":"0"`) || !strings.Contains(param, `"arrGameChoiceNum":null`) {
		t.Errorf("param JSON = %q", param)
	}
	dateRe := regexp.MustCompile(`^\d{4}/\d{2}/\d{2}$`)
	if !dateRe.MatchString(form.Get("ROUND_DRAW_DATE")) {
		t.Errorf("ROUND_DRAW_DATE = %q", form.Get("ROUND_DRAW_DATE"))
	}
	if !dateRe.MatchString(form.Get("WAMT_PAY_TLMT_END_DT")) {
		t.Errorf("WAMT_PAY_TLMT_END_DT = %q", form.Get("WAMT_PAY_TLMT_END_DT"))
	}

	// Required AJAX headers on both ready and execBuy.
	for _, sub := range []string{"egovUserReadySocket", "execBuy"} {
		req, _ := findRequest(stub, sub)
		if req.Header.Get("Origin") != "https://ol.dhlottery.co.kr" ||
			req.Header.Get("Referer") != "https://ol.dhlottery.co.kr/olotto/game/game645.do" ||
			req.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			t.Errorf("%s missing AJAX headers: %v", sub, req.Header)
		}
	}

	// Success notification.
	p := col.Payloads()[0]
	if p.Type != notify.Success || p.Title != "로또 구매 완료" {
		t.Errorf("payload = %+v", p)
	}
	if !strings.Contains(p.Message, "1203회") {
		t.Errorf("message = %q", p.Message)
	}
	if !strings.Contains(detailValue(p, "결제금액"), "5,000") {
		t.Errorf("결제금액 = %q", detailValue(p, "결제금액"))
	}
	if detailValue(p, "잔액") == "" {
		t.Error("잔액 detail missing")
	}
}

func TestPurchaseReadyIPUsed(t *testing.T) {
	client, stub := buyClient(testutil.JSON(`{"direct_yn":"N","ready_ip":"TESTSERVER","ready_time":"0","ready_cnt":"0"}`), execBuySuccess, nil, nil)
	purchaseLottery(client, &notify.Collector{})
	if execBuyForm(t, stub).Get("direct") != "TESTSERVER" {
		t.Error("ready_ip should be used as direct param")
	}
}

func TestPurchaseLimitExceeded(t *testing.T) {
	exec := testutil.JSON(`{"loginYn":"Y","result":{"resultCode":"-7","resultMsg":"[온라인복권 주간 구매한도] 초과되었습니다."}}`)
	client, _ := buyClient(readyOK, exec, nil, nil)
	col := &notify.Collector{}
	out := purchaseLottery(client, col)
	if out.Success || out.Code != "-7" || !strings.Contains(out.Error, "구매한도") {
		t.Errorf("out = %+v", out)
	}
	if col.Payloads()[0].Type != notify.Error || col.Payloads()[0].Title != "로또 구매 실패" {
		t.Errorf("expected error payload: %+v", col.Payloads()[0])
	}
}

func TestPurchaseNetworkErrorOnReady(t *testing.T) {
	client, _ := buyClient(readyOK, execBuySuccess, errors.New("Network error"), nil)
	col := &notify.Collector{}
	out := purchaseLottery(client, col)
	if out.Success || !strings.Contains(out.Error, "Network error") {
		t.Errorf("out = %+v", out)
	}
	if col.Payloads()[0].Type != notify.Error || col.Payloads()[0].Title != "로또 구매 실패" {
		t.Errorf("expected error payload: %+v", col.Payloads()[0])
	}
}

func TestPurchaseNetworkErrorOnExecution(t *testing.T) {
	client, _ := buyClient(readyOK, execBuySuccess, nil, errors.New("Network error"))
	out := purchaseLottery(client, &notify.Collector{})
	if out.Success || !strings.Contains(out.Error, "Network error") {
		t.Errorf("out = %+v", out)
	}
}

func TestPurchaseCollectorSuccess(t *testing.T) {
	client, _ := buyClient(readyOK, execBuySuccess, nil, nil)
	col := &notify.Collector{}
	purchaseLottery(client, col)
	if len(col.Payloads()) != 1 || col.Payloads()[0].Type != notify.Success {
		t.Errorf("expected 1 success payload, got %+v", col.Payloads())
	}
}
