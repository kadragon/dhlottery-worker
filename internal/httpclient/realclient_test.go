package httpclient_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func TestRealClientDoesNotFollowRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.SetCookie(w, &http.Cookie{Name: "DHJSESSIONID", Value: "real-session"})
			w.Header().Set("Location", "/next")
			w.WriteHeader(http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := httpclient.New()
	resp, err := client.Fetch(srv.URL+"/start", httpclient.RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != http.StatusFound {
		t.Errorf("status = %d, want 302 (redirect must not be followed)", resp.Status)
	}
	if client.Cookie("DHJSESSIONID") != "real-session" {
		t.Errorf("cookie not captured: %q", client.Cookie("DHJSESSIONID"))
	}
}

func TestResponseJSON(t *testing.T) {
	client := httpclient.NewWithDoer(&testutil.StubDoer{Handler: testutil.Sequence(testutil.JSON(`{"a":1,"b":"x"}`))})
	resp, err := client.Fetch("https://example.com", httpclient.RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	if err := resp.JSON(&v); err != nil {
		t.Fatal(err)
	}
	if v.A != 1 || v.B != "x" {
		t.Errorf("decoded = %+v", v)
	}
}

func TestSetCookie(t *testing.T) {
	client := httpclient.NewWithDoer(&testutil.StubDoer{Handler: testutil.Sequence(testutil.StubResponse{Status: 200})})
	client.SetCookie("K", "V")
	if client.Cookie("K") != "V" {
		t.Errorf("Cookie(K) = %q", client.Cookie("K"))
	}
	if client.CookieHeader() != "K=V" {
		t.Errorf("CookieHeader = %q", client.CookieHeader())
	}
}
