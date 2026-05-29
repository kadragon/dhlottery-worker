package httpclient_test

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func setCookieResp(values ...string) testutil.StubResponse {
	return testutil.StubResponse{Status: 200, Header: http.Header{"Set-Cookie": values}, Body: "<html></html>"}
}

func newClient(handler func(int, testutil.RecordedRequest) (testutil.StubResponse, error)) (*httpclient.Client, *testutil.StubDoer) {
	doer := &testutil.StubDoer{Handler: handler}
	return httpclient.NewWithDoer(doer), doer
}

func TestCaptureSingleCookie(t *testing.T) {
	client, _ := newClient(testutil.Sequence(setCookieResp("JSESSIONID=abc123; Path=/; HttpOnly")))
	if _, err := client.Fetch("https://example.com", httpclient.RequestOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := client.Cookie("JSESSIONID"); got != "abc123" {
		t.Errorf("JSESSIONID = %q, want abc123", got)
	}
}

func TestPreserveEqualsInCookieValue(t *testing.T) {
	client, _ := newClient(testutil.Sequence(setCookieResp("DHJSESSIONID=abc.def==; Path=/; HttpOnly")))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	if got := client.Cookie("DHJSESSIONID"); got != "abc.def==" {
		t.Errorf("DHJSESSIONID = %q, want abc.def==", got)
	}
}

func TestCaptureMultipleCookies(t *testing.T) {
	client, _ := newClient(testutil.Sequence(setCookieResp("session=xyz789; Path=/", "lang=ko; Path=/")))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	if client.Cookie("session") != "xyz789" || client.Cookie("lang") != "ko" {
		t.Errorf("cookies = %q,%q", client.Cookie("session"), client.Cookie("lang"))
	}
}

func TestNoSetCookie(t *testing.T) {
	client, _ := newClient(testutil.Sequence(testutil.StubResponse{Status: 200, Body: "ok"}))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	if client.CookieHeader() != "" {
		t.Errorf("expected no cookies, got %q", client.CookieHeader())
	}
}

func TestIncludeCookiesInSubsequentRequest(t *testing.T) {
	client, doer := newClient(testutil.Sequence(
		setCookieResp("JSESSIONID=abc123; Path=/"),
		testutil.StubResponse{Status: 200, Body: "ok"},
	))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	client.Fetch("https://example.com/page", httpclient.RequestOptions{})

	if got := doer.Requests[1].Header.Get("Cookie"); got != "JSESSIONID=abc123" {
		t.Errorf("Cookie header = %q, want JSESSIONID=abc123", got)
	}
}

func TestCookieHeaderOrder(t *testing.T) {
	client, doer := newClient(testutil.Sequence(
		setCookieResp("session=xyz; Path=/", "lang=ko; Path=/"),
		testutil.StubResponse{Status: 200, Body: "ok"},
	))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	client.Fetch("https://example.com/page", httpclient.RequestOptions{})

	if got := doer.Requests[1].Header.Get("Cookie"); got != "session=xyz; lang=ko" {
		t.Errorf("Cookie header = %q, want insertion order session=xyz; lang=ko", got)
	}
}

func TestNoCookieHeaderWhenEmpty(t *testing.T) {
	client, doer := newClient(testutil.Sequence(testutil.StubResponse{Status: 200, Body: "ok"}))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	if _, ok := doer.Requests[0].Header["Cookie"]; ok {
		t.Error("Cookie header should be absent when no cookies stored")
	}
}

func TestUpdateExistingCookie(t *testing.T) {
	client, _ := newClient(testutil.Sequence(
		setCookieResp("session=initial; Path=/"),
		setCookieResp("session=updated; Path=/"),
	))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	client.Fetch("https://example.com/update", httpclient.RequestOptions{})
	if client.Cookie("session") != "updated" {
		t.Errorf("session = %q, want updated", client.Cookie("session"))
	}
}

func TestMixedUpdateAndAdd(t *testing.T) {
	client, _ := newClient(testutil.Sequence(
		setCookieResp("session=v1; Path=/"),
		setCookieResp("session=v2; Path=/", "user=john; Path=/"),
	))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	client.Fetch("https://example.com/auth", httpclient.RequestOptions{})
	if client.Cookie("session") != "v2" || client.Cookie("user") != "john" {
		t.Errorf("session=%q user=%q", client.Cookie("session"), client.Cookie("user"))
	}
}

func TestCookieOverwriteWarningHidesValue(t *testing.T) {
	var errBuf bytes.Buffer
	logger.SetWriters(os.Stdout, &errBuf)
	defer logger.SetWriters(os.Stdout, os.Stderr)

	client, _ := newClient(testutil.Sequence(
		setCookieResp("session=abc; Path=/"),
		setCookieResp("session=xyz; Path=/"),
	))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	client.Fetch("https://example.com/page", httpclient.RequestOptions{})

	out := errBuf.String()
	if !strings.Contains(out, "cookie_overwritten") || !strings.Contains(out, `"name":"session"`) {
		t.Errorf("expected overwrite warning with name, got %q", out)
	}
	if strings.Contains(out, "abc") || strings.Contains(out, "xyz") {
		t.Errorf("cookie value must not appear in logs: %q", out)
	}
}

func TestNoWarnSameValue(t *testing.T) {
	var errBuf bytes.Buffer
	logger.SetWriters(os.Stdout, &errBuf)
	defer logger.SetWriters(os.Stdout, os.Stderr)

	client, _ := newClient(testutil.Sequence(
		setCookieResp("session=abc; Path=/"),
		setCookieResp("session=abc; Path=/"),
	))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	client.Fetch("https://example.com/page", httpclient.RequestOptions{})

	if strings.Contains(errBuf.String(), "cookie_overwritten") {
		t.Errorf("should not warn when value unchanged: %q", errBuf.String())
	}
}

func TestClearCookies(t *testing.T) {
	client, _ := newClient(testutil.Sequence(setCookieResp("session=xyz; Path=/")))
	client.Fetch("https://example.com", httpclient.RequestOptions{})
	client.ClearCookies()
	if client.CookieHeader() != "" {
		t.Errorf("expected cookies cleared, got %q", client.CookieHeader())
	}
}

func TestResponseTextEUCKR(t *testing.T) {
	// EUC-KR bytes for "당첨" (0xB4 0xE7 0xC3 0xB7).
	body := string([]byte{0xB4, 0xE7, 0xC3, 0xB7})
	client, _ := newClient(testutil.Sequence(testutil.StubResponse{Status: 200, Body: body}))
	resp, err := client.Fetch("https://example.com", httpclient.RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	text, err := resp.Text("euc-kr")
	if err != nil {
		t.Fatal(err)
	}
	if text != "당첨" {
		t.Errorf("euc-kr decode = %q, want 당첨", text)
	}
}
