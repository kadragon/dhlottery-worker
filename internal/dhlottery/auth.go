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
			"Accept":                       "application/json, text/javascript, */*; q=0.01",
			constants.HeaderContentType:    "application/json;charset=UTF-8",
			constants.HeaderUserAgent:      constants.UserAgent,
			constants.HeaderXRequestedWith: constants.HeaderXRequestedWithValue,
			constants.HeaderReferer:        loginPageURL,
			"ajax":                         "true",
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

	// Manual redirects: a 302 to loginSuccess.do is success.
	if resp.Status >= 300 && resp.Status < 400 {
		if strings.Contains(resp.Header.Get("Location"), redirectWord) {
			return nil
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
	return dherr.NewAuth("Unexpected login response", "AUTH_UNEXPECTED_RESPONSE")
}
