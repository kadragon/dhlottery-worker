package dhlottery

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/kadragon/dhlottery-worker/internal/constants"
	"github.com/kadragon/dhlottery-worker/internal/dherr"
	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
)

// RSA-encrypted authentication flow (2026-01):
//  1. GET /login            -> acquire DHJSESSIONID cookie
//  2. GET selectRsaModulus  -> RSA public key
//  3. POST securityLoginCheck.do with RSA PKCS#1 v1.5 encrypted credentials
const (
	loginPageURL  = "https://www.dhlottery.co.kr/login"
	rsaModulusURL = "https://www.dhlottery.co.kr/login/selectRsaModulus.do"
	loginURL      = "https://www.dhlottery.co.kr/login/securityLoginCheck.do"
	maxRedirects  = 5
	authModule    = "auth"

	// Password-expiry handling (2026-07): when the account password is older
	// than 90 days, a valid login 302-redirects to ExpryPswdNoti instead of
	// loginSuccess.do. The site's "다음에 변경" (defer) button POSTs
	// nxtChngProc.do to record the deferral, then navigates to loginSuccess.do
	// to finalize the authenticated session (which sets the userId cookie).
	pswdExpiryMarker = "ExpryPswdNoti"
	nxtChngURL       = "https://www.dhlottery.co.kr/mbrsrvc/nxtChngProc.do"
	loginSuccessURL  = "https://www.dhlottery.co.kr/login/loginSuccess.do?returnUrl=/main"
)

var (
	alertRe      = regexp.MustCompile(`\$\.alert\(['"]([^'"]+)['"]\)`)
	errorMsgRe   = regexp.MustCompile(`const errorMessage = '([^']+)'`)
	redirectWord = "loginSuccess.do"
)

// fetchWithRedirects follows 3xx redirects manually (the client uses manual
// redirect mode) up to maxRedirects, returning the 200 response.
func fetchWithRedirects(client *httpclient.Client, rawURL string, opts httpclient.RequestOptions, context, errorCode string) (*httpclient.Response, error) {
	currentURL := rawURL
	for redirectCount := 0; redirectCount < maxRedirects; redirectCount++ {
		resp, err := client.Fetch(currentURL, opts)
		if err != nil {
			return nil, err
		}
		if resp.Status == 200 {
			return resp, nil
		}
		if resp.Status >= 300 && resp.Status < 400 {
			location := resp.Header.Get("Location")
			if location == "" {
				return nil, dherr.NewAuth(fmt.Sprintf("%s redirect without Location header (status %d)", context, resp.Status), errorCode)
			}
			next, err := resolveURL(currentURL, location)
			if err != nil {
				return nil, dherr.NewAuth(fmt.Sprintf("%s invalid redirect Location: %s", context, location), errorCode)
			}
			currentURL = next
			logger.Debug(context+" redirect", logger.Fields{
				logger.FieldModule: authModule, logger.FieldStatus: resp.Status, "location": currentURL,
			})
			continue
		}
		return nil, dherr.NewAuth(fmt.Sprintf("%s failed with status %d", context, resp.Status), errorCode)
	}
	return nil, dherr.NewAuth(fmt.Sprintf("%s exceeded maximum redirects (%d)", context, maxRedirects), errorCode)
}

func resolveURL(base, location string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(ref).String(), nil
}

// rsaEncrypt encrypts text with RSA PKCS#1 v1.5 and returns hex, matching the
// site's jsbn.js based implementation.
func rsaEncrypt(text, modulusHex, exponentHex string) (string, error) {
	n, ok := new(big.Int).SetString(modulusHex, 16)
	if !ok {
		return "", dherr.NewAuth("Invalid RSA modulus", "AUTH_RSA_KEY_ERROR")
	}
	e, ok := new(big.Int).SetString(exponentHex, 16)
	if !ok {
		return "", dherr.NewAuth("Invalid RSA exponent", "AUTH_RSA_KEY_ERROR")
	}
	pub := &rsa.PublicKey{N: n, E: int(e.Int64())}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(text)) //nolint:staticcheck // DHLottery site requires PKCS#1 v1.5; cannot migrate to OAEP
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(encrypted), nil
}

func initSession(client *httpclient.Client) error {
	resp, err := fetchWithRedirects(client, loginPageURL, httpclient.RequestOptions{
		Method: http.MethodGet,
		Headers: map[string]string{
			"Accept-Charset":          "UTF-8",
			constants.HeaderUserAgent: constants.UserAgent,
		},
	}, "Session initialization", "AUTH_SESSION_INIT_ERROR")
	if err != nil {
		return dherr.WrapAuth(err, "Session initialization")
	}

	logger.Debug("Session initialized", logger.Fields{logger.FieldModule: authModule, logger.FieldStatus: resp.Status})

	if client.Cookie("DHJSESSIONID") == "" {
		return dherr.WrapAuth(
			dherr.NewAuth("DHJSESSIONID cookie was not set during initialization", "AUTH_SESSION_INIT_ERROR"),
			"Session initialization")
	}
	return nil
}

func fetchRsaKey(client *httpclient.Client) (modulus, exponent string, err error) {
	resp, err := fetchWithRedirects(client, rsaModulusURL, httpclient.RequestOptions{
		Method: http.MethodGet,
		Headers: map[string]string{
			constants.HeaderAccept:         constants.AcceptJSON,
			constants.HeaderContentType:    constants.ContentTypeJSON,
			constants.HeaderUserAgent:      constants.UserAgent,
			constants.HeaderXRequestedWith: constants.HeaderXRequestedWithValue,
			constants.HeaderReferer:        loginPageURL,
			constants.HeaderAjax:           constants.HeaderAjaxValue,
		},
	}, "RSA key fetch", "AUTH_RSA_KEY_ERROR")
	if err != nil {
		return "", "", dherr.WrapAuth(err, "RSA key fetch")
	}

	var data struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RsaModulus     string `json:"rsaModulus"`
			PublicExponent string `json:"publicExponent"`
		} `json:"data"`
	}
	if err := resp.JSON(&data); err != nil {
		return "", "", dherr.WrapAuth(err, "RSA key fetch")
	}
	if data.Data.RsaModulus == "" || data.Data.PublicExponent == "" {
		return "", "", dherr.NewAuth("Invalid RSA key response format", "AUTH_RSA_KEY_ERROR")
	}

	logger.Debug("RSA key fetched", logger.Fields{logger.FieldModule: authModule, "modulusLength": len(data.Data.RsaModulus)})
	return data.Data.RsaModulus, data.Data.PublicExponent, nil
}

// login authenticates with DHLottery.
func login(client *httpclient.Client) error {
	if err := initSession(client); err != nil {
		return err
	}

	modulus, exponent, err := fetchRsaKey(client)
	if err != nil {
		return err
	}

	userID, err := env.Get("USER_ID")
	if err != nil {
		return dherr.WrapAuth(err, "Login")
	}
	password, err := env.Get("PASSWORD")
	if err != nil {
		return dherr.WrapAuth(err, "Login")
	}

	encryptedUserID, err := rsaEncrypt(userID, modulus, exponent)
	if err != nil {
		return dherr.WrapAuth(err, "Login")
	}
	encryptedPassword, err := rsaEncrypt(password, modulus, exponent)
	if err != nil {
		return dherr.WrapAuth(err, "Login")
	}

	form := url.Values{}
	form.Set("userId", encryptedUserID)
	form.Set("userPswdEncn", encryptedPassword)
	form.Set("inpUserId", userID)

	resp, err := client.Fetch(loginURL, httpclient.RequestOptions{
		Method: http.MethodPost,
		Headers: map[string]string{
			constants.HeaderContentType: "application/x-www-form-urlencoded",
			constants.HeaderUserAgent:   constants.UserAgent,
			"Origin":                    "https://www.dhlottery.co.kr",
			constants.HeaderReferer:     loginPageURL,
			"Upgrade-Insecure-Requests": "1",
			"Cache-Control":             "max-age=0",
		},
		Body: form.Encode(),
	})
	if err != nil {
		return dherr.WrapAuth(err, "Login")
	}

	logger.Debug("Login response received", logger.Fields{logger.FieldModule: authModule, logger.FieldStatus: resp.Status})

	// Manual redirects: a 302 to loginSuccess.do is success; a 302 to the
	// password-expiry notice means the credentials are valid but the site is
	// nagging to change a >90-day-old password — defer it and finish the login.
	if resp.Status >= 300 && resp.Status < 400 {
		location := resp.Header.Get("Location")
		if strings.Contains(location, redirectWord) {
			return nil
		}
		if strings.Contains(location, pswdExpiryMarker) {
			return completePasswordExpiryLogin(client, location)
		}
	}

	// userId cookie indicates a successful login.
	if client.Cookie("userId") != "" {
		return nil
	}

	body, err := resp.Text("utf-8")
	if err != nil {
		return dherr.WrapAuth(err, "Login")
	}

	if strings.Contains(body, "isLoggedIn = true") {
		return nil
	}
	if m := alertRe.FindStringSubmatch(body); m != nil {
		return dherr.NewAuth(m[1], "AUTH_INVALID_CREDENTIALS")
	}
	if m := errorMsgRe.FindStringSubmatch(body); m != nil && m[1] != "" {
		return dherr.NewAuth(m[1], "AUTH_INVALID_CREDENTIALS")
	}
	if strings.Contains(body, "isLoggedIn = false") {
		return dherr.NewAuth("아이디 또는 비밀번호가 일치하지 않습니다.", "AUTH_INVALID_CREDENTIALS")
	}
	// Include the status and redirect target so an unrecognized post-login state
	// (e.g. a new interstitial like ExpryPswdNoti was in 2026-07) is diagnosable
	// from the notification alone instead of an opaque "Unexpected" message.
	return dherr.NewAuth(
		fmt.Sprintf("Unexpected login response (status %d, location %q)", resp.Status, resp.Header.Get("Location")),
		"AUTH_UNEXPECTED_RESPONSE")
}

// completePasswordExpiryLogin finishes a login that the site interrupted with a
// password-expiry notice (302 -> /mbrsrvc/ExpryPswdNoti). It mirrors the site's
// "다음에 변경" (defer) button: load the interstitial, POST nxtChngProc.do to
// record the deferral, then GET loginSuccess.do to establish the authenticated
// session. Success is confirmed by the userId cookie.
func completePasswordExpiryLogin(client *httpclient.Client, expiryURL string) error {
	logger.Info("Password expiry notice; deferring change to finish login", logger.Fields{logger.FieldModule: authModule})

	// Load the interstitial the browser renders before deferring.
	if _, err := fetchWithRedirects(client, expiryURL, httpclient.RequestOptions{
		Method: http.MethodGet,
		Headers: map[string]string{
			constants.HeaderUserAgent: constants.UserAgent,
			constants.HeaderReferer:   loginPageURL,
		},
	}, "Password expiry notice", "AUTH_PASSWORD_EXPIRY_ERROR"); err != nil {
		return dherr.WrapAuth(err, "Password expiry notice")
	}

	// Record the deferral ("change later"). Route through fetchWithRedirects so
	// a non-200 answer (session-expiry 3xx, 5xx error page) yields a
	// status-aware error instead of an opaque JSON-decode failure, matching the
	// sibling calls in this file.
	resp, err := fetchWithRedirects(client, nxtChngURL, httpclient.RequestOptions{
		Method: http.MethodPost,
		Headers: map[string]string{
			constants.HeaderAccept:         constants.AcceptJSON,
			constants.HeaderContentType:    constants.ContentTypeJSON,
			constants.HeaderUserAgent:      constants.UserAgent,
			constants.HeaderXRequestedWith: constants.HeaderXRequestedWithValue,
			constants.HeaderReferer:        expiryURL,
			constants.HeaderAjax:           constants.HeaderAjaxValue,
		},
		Body: "{}",
	}, "Password change deferral", "AUTH_PASSWORD_EXPIRY_ERROR")
	if err != nil {
		return dherr.WrapAuth(err, "Password change deferral")
	}

	var data struct {
		Data struct {
			ResultCnt int    `json:"resultCnt"`
			ResultMsg string `json:"resultMsg"`
		} `json:"data"`
	}
	if err := resp.JSON(&data); err != nil {
		return dherr.WrapAuth(err, "Password change deferral")
	}
	if data.Data.ResultCnt <= 0 {
		msg := data.Data.ResultMsg
		if msg == "" {
			msg = "비밀번호 변경 유예가 거부되었습니다. 동행복권 비밀번호를 직접 변경해주세요."
		}
		return dherr.NewAuth(msg, "AUTH_PASSWORD_CHANGE_REQUIRED")
	}

	// Finalize the session; loginSuccess.do sets the userId cookie.
	if _, err := fetchWithRedirects(client, loginSuccessURL, httpclient.RequestOptions{
		Method: http.MethodGet,
		Headers: map[string]string{
			constants.HeaderUserAgent: constants.UserAgent,
			constants.HeaderReferer:   expiryURL,
		},
	}, "Login finalization", "AUTH_SESSION_FINALIZE_ERROR"); err != nil {
		return dherr.WrapAuth(err, "Login finalization")
	}

	if client.Cookie("userId") == "" {
		return dherr.NewAuth("Login finalization did not establish an authenticated session", "AUTH_SESSION_FINALIZE_ERROR")
	}
	return nil
}
