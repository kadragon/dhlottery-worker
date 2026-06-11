package dhlottery

import (
	"strings"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/notify"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func facadeClient(handler func(int, testutil.RecordedRequest) (testutil.StubResponse, error)) *Client {
	return &Client{
		http:      httpclient.NewWithDoer(&testutil.StubDoer{Handler: handler}),
		collector: &notify.Collector{},
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c.Collector() == nil {
		t.Error("NewClient should provide a collector")
	}
}

func TestClientLogin(t *testing.T) {
	t.Setenv("USER_ID", "u")
	t.Setenv("PASSWORD", "p")
	c := facadeClient(testutil.Sequence(sessionResp(), rsaResp(), loginSuccessResp()))
	if err := c.Login(); err != nil {
		t.Errorf("Login: %v", err)
	}
}

func TestClientGetAccountInfo(t *testing.T) {
	c := facadeClient(testutil.Sequence(testutil.JSON(roundJSON), testutil.JSON(balanceJSON(20000))))
	info, err := c.GetAccountInfo()
	if err != nil || info.Balance != 20000 || info.CurrentRound != 1206 {
		t.Errorf("GetAccountInfo = %+v, %v", info, err)
	}
}

func TestClientCheckDeposit(t *testing.T) {
	c := facadeClient(testutil.Sequence(testutil.JSON(roundJSON), testutil.JSON(balanceJSON(50000))))
	ok, err := c.CheckDeposit(constants.WeeklyCombinedRequiredBalance)
	if err != nil || !ok {
		t.Errorf("CheckDeposit = %v, %v", ok, err)
	}
}

func TestClientBuy(t *testing.T) {
	c := facadeClient(buyHandler(readyOK, execBuySuccess, nil, nil))
	if out := c.Buy(); !out.Success {
		t.Errorf("Buy = %+v", out)
	}
}

func TestClientCheckWinning(t *testing.T) {
	c := facadeClient(testutil.Sequence(testutil.StubResponse{
		Status: 200,
		Body:   ledgerFixture(t),
	}))
	results := c.CheckWinning(parseTime(t, "2025-12-15T10:00:00+09:00"))
	if len(results) != 3 {
		t.Errorf("CheckWinning = %+v", results)
	}
}

func TestClientAggregateLedger(t *testing.T) {
	c := facadeClient(testutil.Sequence(testutil.StubResponse{
		Status: 200,
		Body:   ledgerFixture(t),
	}))
	s, ok := c.AggregateLedger("20260401", parseTime(t, "2026-06-08T10:00:00+09:00"))
	if !ok || s.CumulativePurchase != 14000 || s.CumulativeWinning != 2001005000 {
		t.Errorf("AggregateLedger = %+v, ok=%v", s, ok)
	}
}

func TestClientReservePension(t *testing.T) {
	stub := &testutil.StubDoer{Handler: func(_ int, r testutil.RecordedRequest) (testutil.StubResponse, error) {
		switch {
		case strings.Contains(r.URL, "TotalGame.jsp"), strings.Contains(r.URL, "reserveGame.jsp"):
			return testutil.JSON("{}"), nil
		case strings.Contains(r.URL, "roundRemainTime.do"):
			return testutil.JSON(`{"resultCode":"100","ROUND":"303","DRAW_DATE":"2026-02-19"}`), nil
		case strings.Contains(r.URL, "checkDeposit.do"):
			return encResp(t, `{"resultCode":"100","deposit":"10000"}`), nil
		case strings.Contains(r.URL, "checkMyReserve.do"):
			return encResp(t, `{"resultCode":"100","doubleRound":[]}`), nil
		case strings.Contains(r.URL, "addMyReserve.do"):
			return encResp(t, `{"resultCode":"100","resultMsg":"예약 성공","reserveOrderNo":"A-1"}`), nil
		}
		return testutil.StubResponse{}, nil
	}}
	c := &Client{http: httpclient.NewWithDoer(stub), collector: &notify.Collector{}}
	c.http.SetCookie("JSESSIONID", pensionSession)

	out := c.ReservePensionNextWeek()
	if out.Status != "success" || out.TargetRound != 304 {
		t.Errorf("ReservePensionNextWeek = %+v", out)
	}
}
