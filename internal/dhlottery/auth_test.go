package dhlottery

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/dherr"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

// Valid 2048-bit RSA modulus (512 hex chars) from the original test suite.
const testRSAModulus = "9e3076fbc2019ec646dbe2a6d0d22ba2241d85ea363b77f583a9ca0646fae3032d5100a41f67794ca5ff80fc360f9429a0a075be9ef3a1bca4e8bea41915a4491da0e4afdc4081bf3500a502717d913638daaf4883c25561c4f5937fe86478dca94f65624a7de907a7f363994593a69346d90e8f8f8a709cb61936b8bad07d4e29343f11fafa62245662480a4b8206858e4743aea6f410f1d1ad85dafffe77dd2301e0dd5a5b0d5c7f63795cbdc8cd7bc4c90c9f0e6e534227c0f696957badad1b5af3f6722ee99d166f5cf574ae2524763e54590cbd4c9a57ab4d6851bb1870d45bc7439b7805e4be7ae7eca2ff66e7f3186fd21adda6f76efb9b5af8fa01e9"

func sessionResp() testutil.StubResponse {
	return testutil.StubResponse{
		Status: 200,
		Header: http.Header{"Set-Cookie": {"DHJSESSIONID=mock-session-id; Path=/"}},
		Body:   "<!DOCTYPE html>...",
	}
}

func rsaResp() testutil.StubResponse {
	return testutil.JSON(`{"code":"0000","msg":"success","data":{"rsaModulus":"` + testRSAModulus + `","publicExponent":"10001"}}`)
}

func loginSuccessResp() testutil.StubResponse {
	return testutil.StubResponse{Status: 200, Body: `<script>const isLoggedIn = true;</script>`}
}

func loginFailResp(msg string) testutil.StubResponse {
	return testutil.StubResponse{Status: 200, Body: `<script>const isLoggedIn = false; const errorMessage = '` + msg + `';</script>`}
}

func authClient(t *testing.T, handler func(int, testutil.RecordedRequest) (testutil.StubResponse, error)) (*httpclient.Client, *testutil.StubDoer) {
	t.Helper()
	t.Setenv("USER_ID", "testuser")
	t.Setenv("PASSWORD", "testpass123")
	stub := &testutil.StubDoer{Handler: handler}
	return httpclient.NewWithDoer(stub), stub
}

func TestLoginSuccess(t *testing.T) {
	client, stub := authClient(t, testutil.Sequence(sessionResp(), rsaResp(), loginSuccessResp()))
	if err := login(client); err != nil {
		t.Fatalf("login: %v", err)
	}
	if len(stub.Requests) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(stub.Requests))
	}
	if stub.Requests[0].URL != loginPageURL {
		t.Errorf("req[0] = %s, want %s", stub.Requests[0].URL, loginPageURL)
	}
	if stub.Requests[1].URL != rsaModulusURL {
		t.Errorf("req[1] = %s, want %s", stub.Requests[1].URL, rsaModulusURL)
	}
	if stub.Requests[2].URL != loginURL || stub.Requests[2].Method != http.MethodPost {
		t.Errorf("req[2] = %s %s", stub.Requests[2].Method, stub.Requests[2].URL)
	}
}

func TestLoginRequestFormat(t *testing.T) {
	t.Setenv("USER_ID", "test@email.com")
	t.Setenv("PASSWORD", "p@ss w0rd!")
	stub := &testutil.StubDoer{Handler: testutil.Sequence(sessionResp(), rsaResp(), loginSuccessResp())}
	client := httpclient.NewWithDoer(stub)
	if err := login(client); err != nil {
		t.Fatalf("login: %v", err)
	}

	body := stub.Requests[2].Body
	for _, field := range []string{"userId=", "userPswdEncn=", "inpUserId="} {
		if !strings.Contains(body, field) {
			t.Errorf("body missing %q: %s", field, body)
		}
	}
	if !strings.Contains(body, "inpUserId=test%40email.com") {
		t.Errorf("inpUserId not URL-encoded plaintext: %s", body)
	}
	if strings.Contains(body, "p%40ss") {
		t.Errorf("plaintext password leaked into body: %s", body)
	}
	if ct := stub.Requests[2].Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", ct)
	}
	if origin := stub.Requests[2].Header.Get("Origin"); origin != "https://www.dhlottery.co.kr" {
		t.Errorf("Origin = %q", origin)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(sessionResp(), rsaResp(), loginFailResp("인증 실패")))
	err := login(client)
	if err == nil {
		t.Fatal("expected error")
	}
	if e, ok := err.(*dherr.Error); !ok || e.Kind != dherr.KindAuth {
		t.Errorf("expected auth error, got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "인증 실패") {
		t.Errorf("error should carry the server message: %v", err)
	}
}

func TestLoginUnexpectedResponse(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(sessionResp(), rsaResp(),
		testutil.StubResponse{Status: 200, Body: "Unexpected response without isLoggedIn"}))
	err := login(client)
	if err == nil || dherr.Code(err) != "AUTH_UNEXPECTED_RESPONSE" {
		t.Errorf("expected AUTH_UNEXPECTED_RESPONSE, got %v", err)
	}
}

func pswdExpiryRedirectResp() testutil.StubResponse {
	return testutil.StubResponse{Status: 302, Header: http.Header{"Location": {"https://www.dhlottery.co.kr/mbrsrvc/ExpryPswdNoti"}}}
}

// TestLoginPasswordExpiryDeferSuccess reproduces the 2026-07 breakage: a valid
// login is interrupted by the >90-day password-expiry notice (302 ->
// /mbrsrvc/ExpryPswdNoti). The client must defer the change (POST
// nxtChngProc.do) and finalize the session (GET loginSuccess.do), which sets the
// userId cookie.
func TestLoginPasswordExpiryDeferSuccess(t *testing.T) {
	client, stub := authClient(t, testutil.Sequence(
		sessionResp(), rsaResp(),
		pswdExpiryRedirectResp(), // login POST -> 302 ExpryPswdNoti
		testutil.StubResponse{Status: 200, Body: "<html>expiry notice</html>"}, // GET interstitial
		testutil.JSON(`{"data":{"resultCnt":1}}`),                              // nxtChngProc.do -> defer OK
		testutil.StubResponse{ // loginSuccess.do finalizes session, sets userId cookie
			Status: 200,
			Header: http.Header{"Set-Cookie": {"userId=testuser; Path=/"}},
			Body:   "ok",
		},
	))
	if err := login(client); err != nil {
		t.Fatalf("password-expiry login should succeed via defer: %v", err)
	}
	if len(stub.Requests) != 6 {
		t.Fatalf("expected 6 requests, got %d", len(stub.Requests))
	}
	if stub.Requests[4].URL != nxtChngURL || stub.Requests[4].Method != http.MethodPost {
		t.Errorf("req[4] = %s %s, want POST %s", stub.Requests[4].Method, stub.Requests[4].URL, nxtChngURL)
	}
	if !strings.Contains(stub.Requests[5].URL, "loginSuccess.do") {
		t.Errorf("req[5] = %s, want loginSuccess.do", stub.Requests[5].URL)
	}
}

// TestLoginPasswordExpiryDeferRejected covers the site refusing the deferral
// (resultCnt <= 0) — e.g. a KISA-mandated forced change — which must surface an
// actionable AUTH_PASSWORD_CHANGE_REQUIRED error carrying the server message.
func TestLoginPasswordExpiryDeferRejected(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(
		sessionResp(), rsaResp(),
		pswdExpiryRedirectResp(),
		testutil.StubResponse{Status: 200, Body: "<html>expiry notice</html>"},
		testutil.JSON(`{"data":{"resultCnt":0,"resultMsg":"비밀번호를 변경해야 합니다."}}`),
	))
	err := login(client)
	if err == nil || dherr.Code(err) != "AUTH_PASSWORD_CHANGE_REQUIRED" {
		t.Fatalf("expected AUTH_PASSWORD_CHANGE_REQUIRED, got %v", err)
	}
	if !strings.Contains(err.Error(), "비밀번호를 변경해야 합니다.") {
		t.Errorf("error should carry the server resultMsg: %v", err)
	}
}

// TestLoginPasswordExpiryNoUserIdCookie guards the finalization: if
// loginSuccess.do does not establish the userId cookie, login must fail rather
// than silently returning a half-authenticated session.
func TestLoginPasswordExpiryNoUserIdCookie(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(
		sessionResp(), rsaResp(),
		pswdExpiryRedirectResp(),
		testutil.StubResponse{Status: 200, Body: "<html>expiry notice</html>"},
		testutil.JSON(`{"data":{"resultCnt":1}}`),
		testutil.StubResponse{Status: 200, Body: "no cookie here"},
	))
	if err := login(client); err == nil || dherr.Code(err) != "AUTH_SESSION_FINALIZE_ERROR" {
		t.Errorf("expected AUTH_SESSION_FINALIZE_ERROR, got %v", err)
	}
}

func TestLoginNon200OnLogin(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(sessionResp(), rsaResp(),
		testutil.StubResponse{Status: 500, Body: "Server Error"}))
	if err := login(client); err == nil {
		t.Error("expected error on 500 login response")
	}
}

func TestLoginNon200OnSessionInit(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(testutil.StubResponse{Status: 500, Body: "Server Error"}))
	err := login(client)
	if err == nil || dherr.Code(err) != "AUTH_SESSION_INIT_ERROR" {
		t.Errorf("expected AUTH_SESSION_INIT_ERROR, got %v", err)
	}
}

func TestLogin302Success(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(sessionResp(), rsaResp(),
		testutil.StubResponse{Status: 302, Header: http.Header{"Location": {"/login/loginSuccess.do"}}}))
	if err := login(client); err != nil {
		t.Errorf("302 to loginSuccess.do should succeed: %v", err)
	}
}

func TestLogin302NonSuccess(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(sessionResp(), rsaResp(),
		testutil.StubResponse{Status: 302, Header: http.Header{"Location": {"/login/loginFail.do"}}}))
	if err := login(client); err == nil {
		t.Error("302 to non-success location should fail")
	}
}

func TestLoginUserIdCookieSuccess(t *testing.T) {
	// Login response is 200 with no body indicator, but sets a userId cookie.
	client, _ := authClient(t, testutil.Sequence(sessionResp(), rsaResp(),
		testutil.StubResponse{
			Status: 200,
			Header: http.Header{"Set-Cookie": {"userId=testuser; Path=/"}},
			Body:   "<html><body>No login indicator here</body></html>",
		}))
	if err := login(client); err != nil {
		t.Errorf("userId cookie should indicate success: %v", err)
	}
}

func TestLoginFollowsRedirectDuringSessionInit(t *testing.T) {
	client, stub := authClient(t, testutil.Sequence(
		testutil.StubResponse{Status: 301, Header: http.Header{"Location": {"https://www.dhlottery.co.kr/login.do"}}},
		sessionResp(),
		rsaResp(),
		loginSuccessResp(),
	))
	if err := login(client); err != nil {
		t.Errorf("should follow 301 during session init: %v", err)
	}
	if len(stub.Requests) != 4 {
		t.Errorf("expected 4 requests, got %d", len(stub.Requests))
	}
}

func TestLoginFollowsRedirectDuringRsaFetch(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(
		sessionResp(),
		testutil.StubResponse{Status: 301, Header: http.Header{"Location": {"https://www.dhlottery.co.kr/login/selectRsaModulus2.do"}}},
		rsaResp(),
		loginSuccessResp(),
	))
	if err := login(client); err != nil {
		t.Errorf("should follow 301 during RSA fetch: %v", err)
	}
}

func TestLoginRsaFetchFailure(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(sessionResp(),
		testutil.StubResponse{Status: 500, Body: "Server Error"}))
	if err := login(client); err == nil {
		t.Error("expected error on RSA fetch failure")
	}
}

func TestLoginInvalidRsaResponse(t *testing.T) {
	client, _ := authClient(t, testutil.Sequence(sessionResp(), testutil.JSON(`{"data":{}}`)))
	err := login(client)
	if err == nil || dherr.Code(err) != "AUTH_RSA_KEY_ERROR" {
		t.Errorf("expected AUTH_RSA_KEY_ERROR, got %v", err)
	}
}
