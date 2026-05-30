package dhlottery

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/dherr"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

const roundJSON = `{"resultCode":null,"resultMessage":null,"data":{"result":{"ltEpsd":1206,"ltRflYmd":"20260110","ltRflHh":"20","ltRflMm":"00"},"gameMng":null}}`

func balanceFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "selectUserMndp-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func accountClient(handler func(int, testutil.RecordedRequest) (testutil.StubResponse, error)) *httpclient.Client {
	return httpclient.NewWithDoer(&testutil.StubDoer{Handler: handler})
}

func TestGetAccountInfoSuccess(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.JSON(roundJSON), testutil.JSON(balanceFixture(t))))
	info, err := getAccountInfo(client)
	if err != nil {
		t.Fatalf("getAccountInfo: %v", err)
	}
	if info.CurrentRound != 1206 {
		t.Errorf("CurrentRound = %d, want 1206", info.CurrentRound)
	}
	if info.Balance != 20000 {
		t.Errorf("Balance = %d, want 20000", info.Balance)
	}
}

func TestGetAccountInfoURLsAndMethod(t *testing.T) {
	stub := &testutil.StubDoer{Handler: testutil.Sequence(testutil.JSON(roundJSON), testutil.JSON(balanceFixture(t)))}
	client := httpclient.NewWithDoer(stub)
	if _, err := getAccountInfo(client); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stub.Requests[0].URL, "/lt645/selectThsLt645Info.do") {
		t.Errorf("req[0] = %s", stub.Requests[0].URL)
	}
	if !strings.Contains(stub.Requests[1].URL, "/mypage/selectUserMndp.do") {
		t.Errorf("req[1] = %s", stub.Requests[1].URL)
	}
	if stub.Requests[0].Method != http.MethodGet {
		t.Errorf("round API method = %s, want GET", stub.Requests[0].Method)
	}
}

func TestGetAccountInfoBalanceValues(t *testing.T) {
	cases := []struct {
		name string
		json string
		want int
	}{
		{"explicit", `{"data":{"userMndp":{"crntEntrsAmt":5000}}}`, 5000},
		{"zero", `{"data":{"userMndp":{"crntEntrsAmt":0}}}`, 0},
		{"large", `{"data":{"userMndp":{"crntEntrsAmt":1234567}}}`, 1234567},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := accountClient(testutil.Sequence(testutil.JSON(roundJSON), testutil.JSON(c.json)))
			info, err := getAccountInfo(client)
			if err != nil {
				t.Fatal(err)
			}
			if info.Balance != c.want {
				t.Errorf("Balance = %d, want %d", info.Balance, c.want)
			}
		})
	}
}

func TestGetAccountInfoDifferentRound(t *testing.T) {
	client := accountClient(testutil.Sequence(
		testutil.JSON(`{"data":{"result":{"ltEpsd":998}}}`),
		testutil.JSON(balanceFixture(t)),
	))
	info, err := getAccountInfo(client)
	if err != nil {
		t.Fatal(err)
	}
	if info.CurrentRound != 998 {
		t.Errorf("CurrentRound = %d, want 998", info.CurrentRound)
	}
}

func TestGetAccountInfoHTTPError(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.StubResponse{Status: 500, Body: "Error"}))
	if _, err := getAccountInfo(client); err == nil {
		t.Error("expected error on HTTP 500")
	} else if _, ok := err.(*dherr.Error); !ok {
		t.Errorf("expected *dherr.Error, got %T", err)
	}
}

func TestGetAccountInfoRoundRedirect(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.StubResponse{
		Status: 302,
		Header: http.Header{"Location": {"https://www.dhlottery.co.kr/"}},
	}))
	_, err := getAccountInfo(client)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "round API") || !strings.Contains(msg, "HTTP 302") || !strings.Contains(msg, "Location:") {
		t.Errorf("error should describe round API redirect with location: %q", msg)
	}
}

func TestGetAccountInfoBalanceRedirect(t *testing.T) {
	client := accountClient(testutil.Sequence(
		testutil.JSON(roundJSON),
		testutil.StubResponse{Status: 302, Header: http.Header{"Location": {"https://www.dhlottery.co.kr/login"}}},
	))
	_, err := getAccountInfo(client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "balance API") || !strings.Contains(err.Error(), "HTTP 302") {
		t.Errorf("error should describe balance API redirect: %q", err.Error())
	}
}

func TestGetAccountInfoMissingBalance(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.JSON(roundJSON), testutil.JSON(`{"data":{"userMndp":{}}}`)))
	if _, err := getAccountInfo(client); err == nil {
		t.Error("expected error when crntEntrsAmt missing")
	}
}

func TestGetAccountInfoMissingRound(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.JSON(`{}`)))
	_, err := getAccountInfo(client)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "round") {
		t.Errorf("expected round error, got %v", err)
	}
}

func TestGetAccountInfoNegativeBalance(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.JSON(roundJSON), testutil.JSON(`{"data":{"userMndp":{"crntEntrsAmt":-1000}}}`)))
	if _, err := getAccountInfo(client); err == nil {
		t.Error("expected error on negative balance")
	}
}

func TestGetAccountInfoZeroRound(t *testing.T) {
	client := accountClient(testutil.Sequence(testutil.JSON(`{"data":{"result":{"ltEpsd":0}}}`)))
	if _, err := getAccountInfo(client); err == nil {
		t.Error("expected error on zero round")
	}
}
