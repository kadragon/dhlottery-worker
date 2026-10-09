package checkpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

// sleeps records the backoffs requested by the code under test.
var sleeps []time.Duration

func install(t *testing.T, h func(int, testutil.RecordedRequest) (testutil.StubResponse, error)) *testutil.StubDoer {
	t.Helper()
	stub := &testutil.StubDoer{Handler: h}
	orig, origSleep := doer, sleep
	sleeps = nil
	doer, sleep = stub, func(d time.Duration) { sleeps = append(sleeps, d) }
	t.Cleanup(func() { doer, sleep = orig, origSleep })
	return stub
}

func configure(t *testing.T) {
	t.Helper()
	t.Setenv("GIST_TOKEN", "tok")
	t.Setenv("GIST_ID", "abc123")
}

func gistBody(t *testing.T, content string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"files": map[string]any{fileName: map[string]any{"content": content}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestLoadValid(t *testing.T) {
	configure(t)
	stub := install(t, testutil.Sequence(testutil.JSON(gistBody(t,
		`{"version":1,"start":"2020-01-01","through":"2026-08-23","purchase":1000,"winning":500}`))))

	cp := Load()
	want := Checkpoint{Version: schemaVersion, Start: "2020-01-01", Through: "2026-08-23", Purchase: 1000, Winning: 500}
	if cp == nil || *cp != want {
		t.Fatalf("Load = %+v, want %+v", cp, want)
	}
	req := stub.Requests[0]
	if req.Method != "GET" || req.URL != "https://api.github.com/gists/abc123" {
		t.Errorf("request = %s %s", req.Method, req.URL)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer tok" {
		t.Errorf("Authorization = %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != "dhlottery-worker" {
		t.Errorf("User-Agent = %q", got)
	}
}

func TestLoadUnconfigured(t *testing.T) {
	t.Setenv("GIST_TOKEN", "")
	t.Setenv("GIST_ID", "")
	stub := install(t, testutil.Sequence(testutil.JSON("{}")))
	if cp := Load(); cp != nil {
		t.Errorf("Load = %+v, want nil", cp)
	}
	if len(stub.Requests) != 0 {
		t.Errorf("expected no request when unconfigured, got %d", len(stub.Requests))
	}
}

func TestLoadFailuresReturnNil(t *testing.T) {
	cases := map[string]func(int, testutil.RecordedRequest) (testutil.StubResponse, error){
		"network": func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
			return testutil.StubResponse{}, errors.New("boom")
		},
		"non-200":      testutil.Sequence(testutil.StubResponse{Status: 404, Body: "{}"}),
		"bad gist":     testutil.Sequence(testutil.JSON("not json")),
		"missing file": testutil.Sequence(testutil.JSON(`{"files":{"other.json":{"content":"{}"}}}`)),
		"bad content":  testutil.Sequence(testutil.JSON(gistBody(t, "garbage"))),
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			configure(t)
			install(t, h)
			if cp := Load(); cp != nil {
				t.Errorf("Load = %+v, want nil", cp)
			}
		})
	}
}

// A transport error must not leak the secret gist ID (embedded in the URL)
// into logs.
func TestLoadNetworkErrorRedactsGistID(t *testing.T) {
	configure(t)
	var logged bytes.Buffer
	logger.SetWriters(&logged, &logged)
	t.Cleanup(func() { logger.SetWriters(os.Stdout, os.Stderr) })
	install(t, func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
		return testutil.StubResponse{}, errors.New("dial tcp: timeout")
	})
	// A real *http.Client wraps transport failures in *url.Error carrying the URL.
	orig := doer
	doer = urlErrDoer{inner: orig}

	if cp := Load(); cp != nil {
		t.Fatalf("Load = %+v, want nil", cp)
	}
	if strings.Contains(logged.String(), "abc123") {
		t.Errorf("log leaks gist ID: %s", logged.String())
	}
	if !strings.Contains(logged.String(), "dial tcp: timeout") {
		t.Errorf("log lost the cause: %s", logged.String())
	}
}

type urlErrDoer struct {
	inner interface {
		Do(*http.Request) (*http.Response, error)
	}
}

func (d urlErrDoer) Do(req *http.Request) (*http.Response, error) {
	resp, err := d.inner.Do(req)
	if err != nil {
		return nil, &url.Error{Op: req.Method, URL: req.URL.String(), Err: err}
	}
	return resp, nil
}

func TestSave(t *testing.T) {
	configure(t)
	stub := install(t, testutil.Sequence(testutil.JSON("{}")))
	cp := Checkpoint{Start: "2020-01-01", Through: "2026-08-23", Purchase: 1000, Winning: 500}
	if !Save(cp) {
		t.Fatal("Save = false, want true")
	}
	req := stub.Requests[0]
	if req.Method != "PATCH" || req.URL != "https://api.github.com/gists/abc123" {
		t.Errorf("request = %s %s", req.Method, req.URL)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer tok" {
		t.Errorf("Authorization = %q", got)
	}
	var body struct {
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	want := cp
	want.Version = schemaVersion // Save stamps the current schema version
	var got Checkpoint
	if err := json.Unmarshal([]byte(body.Files[fileName].Content), &got); err != nil || got != want {
		t.Errorf("saved content = %+v (err %v), want %+v", got, err, want)
	}
}

func TestSaveFailures(t *testing.T) {
	cases := map[string]func(int, testutil.RecordedRequest) (testutil.StubResponse, error){
		"network": func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
			return testutil.StubResponse{}, errors.New("boom")
		},
		"non-2xx": testutil.Sequence(testutil.StubResponse{Status: 403, Body: "{}"}),
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			configure(t)
			install(t, h)
			if Save(Checkpoint{}) {
				t.Error("Save = true, want false")
			}
		})
	}
}

func TestSaveUnconfigured(t *testing.T) {
	t.Setenv("GIST_TOKEN", "")
	t.Setenv("GIST_ID", "")
	stub := install(t, testutil.Sequence(testutil.JSON("{}")))
	if Save(Checkpoint{}) {
		t.Error("Save = true, want false")
	}
	if len(stub.Requests) != 0 {
		t.Errorf("expected no request when unconfigured, got %d", len(stub.Requests))
	}
}

// A checkpoint written under a different aggregation schema (or before
// versioning existed) must not seed an incremental run.
func TestLoadRejectsSchemaVersionMismatch(t *testing.T) {
	cases := map[string]string{
		"missing": `{"start":"2020-01-01","through":"2026-08-23","purchase":1000,"winning":500}`,
		"other":   `{"version":99,"start":"2020-01-01","through":"2026-08-23","purchase":1000,"winning":500}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			configure(t)
			install(t, testutil.Sequence(testutil.JSON(gistBody(t, content))))
			if cp := Load(); cp != nil {
				t.Errorf("Load = %+v, want nil", cp)
			}
		})
	}
}

var errTransient = errors.New("connection reset")

// transientCases maps subtest names to a first-call failure: 0 means a
// transport error, otherwise that HTTP status.
var transientCases = map[string]int{"transport": 0, "408": 408, "429": 429, "500": 500, "502": 502, "503": 503, "504": 504}

// transientThen fails the first call (transport error when status is 0,
// otherwise that status) and answers ok afterwards.
func transientThen(status int, ok testutil.StubResponse) func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
	return func(call int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
		if call > 0 {
			return ok, nil
		}
		if status == 0 {
			return testutil.StubResponse{}, errTransient
		}
		return testutil.StubResponse{Status: status, Body: "{}"}, nil
	}
}

func TestLoadRetriesTransientFailureOnce(t *testing.T) {
	valid := testutil.JSON(gistBody(t, `{"version":1,"start":"2020-01-01","through":"2026-08-23","purchase":1000,"winning":500}`))
	for name, status := range transientCases {
		t.Run(name, func(t *testing.T) {
			configure(t)
			stub := install(t, transientThen(status, valid))
			if cp := Load(); cp == nil {
				t.Fatal("Load = nil, want checkpoint after retry")
			}
			if len(stub.Requests) != 2 {
				t.Errorf("requests = %d, want 2", len(stub.Requests))
			}
			if len(sleeps) != 1 || sleeps[0] != retryDelay {
				t.Errorf("sleeps = %v, want [%v]", sleeps, retryDelay)
			}
		})
	}
}

func TestSaveRetriesTransientFailureOnce(t *testing.T) {
	for name, status := range transientCases {
		t.Run(name, func(t *testing.T) {
			configure(t)
			stub := install(t, transientThen(status, testutil.JSON("{}")))
			if !Save(Checkpoint{Start: "2020-01-01", Through: "2026-08-23"}) {
				t.Fatal("Save = false, want true after retry")
			}
			if len(stub.Requests) != 2 {
				t.Fatalf("requests = %d, want 2", len(stub.Requests))
			}
			if len(sleeps) != 1 || sleeps[0] != retryDelay {
				t.Errorf("sleeps = %v, want [%v]", sleeps, retryDelay)
			}
			if stub.Requests[1].Body == "" || stub.Requests[1].Body != stub.Requests[0].Body {
				t.Errorf("retry body = %q, want same as first %q", stub.Requests[1].Body, stub.Requests[0].Body)
			}
		})
	}
}

func TestRetryGivesUpAfterSecondAttempt(t *testing.T) {
	configure(t)
	stub := install(t, testutil.Sequence(testutil.StubResponse{Status: 503, Body: "{}"}))
	if cp := Load(); cp != nil {
		t.Errorf("Load = %+v, want nil", cp)
	}
	if Save(Checkpoint{}) {
		t.Error("Save = true, want false")
	}
	if len(stub.Requests) != 4 {
		t.Errorf("requests = %d, want 4 (2 per call)", len(stub.Requests))
	}
}

func TestNoRetryOnPermanentStatus(t *testing.T) {
	configure(t)
	stub := install(t, testutil.Sequence(testutil.StubResponse{Status: 404, Body: "{}"}))
	if cp := Load(); cp != nil {
		t.Errorf("Load = %+v, want nil", cp)
	}
	if Save(Checkpoint{}) {
		t.Error("Save = true, want false")
	}
	if len(stub.Requests) != 2 {
		t.Errorf("requests = %d, want 2 (1 per call)", len(stub.Requests))
	}
	if len(sleeps) != 0 {
		t.Errorf("sleeps = %v, want none for a permanent status", sleeps)
	}
}

func TestRateLimitRetryHonorsServerWait(t *testing.T) {
	fixed := time.Unix(1_700_000_000, 0)
	origNow := now
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = origNow })

	cases := map[string]struct {
		status   int
		header   http.Header
		requests int
		sleeps   []time.Duration
	}{
		"429 Retry-After":           {429, http.Header{"Retry-After": {"3"}}, 2, []time.Duration{3 * time.Second}},
		"403 Retry-After":           {403, http.Header{"Retry-After": {"3"}}, 2, []time.Duration{3 * time.Second}},
		"403 remaining 0 reset":     {403, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"1700000004"}}, 2, []time.Duration{5 * time.Second}},
		"403 remaining 0 past":      {403, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"1699999990"}}, 2, []time.Duration{resetBuffer}},
		"403 remaining 0 no reset":  {403, http.Header{"X-Ratelimit-Remaining": {"0"}}, 1, nil},
		"429 no headers":            {429, nil, 2, []time.Duration{retryDelay}},
		"429 Retry-After over cap":  {429, http.Header{"Retry-After": {"60"}}, 1, nil},
		"403 reset over cap":        {403, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"1700003600"}}, 1, nil},
		"403 without rate limiting": {403, nil, 1, nil},
		"403 remaining nonzero":     {403, http.Header{"X-Ratelimit-Remaining": {"10"}}, 1, nil},
	}
	valid := testutil.JSON(gistBody(t, `{"version":1,"start":"2020-01-01","through":"2026-08-23","purchase":1000,"winning":500}`))
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			configure(t)
			stub := install(t, func(call int, _ testutil.RecordedRequest) (testutil.StubResponse, error) {
				if call > 0 {
					return valid, nil
				}
				return testutil.StubResponse{Status: tc.status, Header: tc.header, Body: "{}"}, nil
			})
			cp := Load()
			if len(stub.Requests) != tc.requests {
				t.Errorf("requests = %d, want %d", len(stub.Requests), tc.requests)
			}
			if (cp != nil) != (tc.requests == 2) {
				t.Errorf("Load = %+v, want checkpoint only after a retry", cp)
			}
			if len(sleeps) != len(tc.sleeps) || (len(sleeps) > 0 && sleeps[0] != tc.sleeps[0]) {
				t.Errorf("sleeps = %v, want %v", sleeps, tc.sleeps)
			}
		})
	}
}

func TestSaveUndelivered(t *testing.T) {
	configure(t)
	var errBuf bytes.Buffer
	logger.SetWriters(&errBuf, &errBuf)
	defer logger.SetWriters(os.Stdout, os.Stderr)

	stub := install(t, testutil.Sequence(testutil.JSON("{}")))
	content := "생성: 2026-10-12 10:00 KST\n\n✅ **잔액 12345**"
	if !SaveUndelivered(content) {
		t.Fatal("SaveUndelivered = false, want true")
	}
	if len(stub.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(stub.Requests))
	}
	req := stub.Requests[0]
	if req.Method != "PATCH" || req.URL != "https://api.github.com/gists/abc123" {
		t.Errorf("request = %s %s", req.Method, req.URL)
	}
	var body struct {
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if len(body.Files) != 1 {
		t.Errorf("files = %v, want only %s", body.Files, undeliveredFileName)
	}
	if got := body.Files[undeliveredFileName].Content; got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
	if logs := errBuf.String(); strings.Contains(logs, "12345") || strings.Contains(logs, "abc123") {
		t.Errorf("logs leak content or gist ID: %q", logs)
	}
}

func TestSaveUndeliveredFailures(t *testing.T) {
	cases := map[string]func(int, testutil.RecordedRequest) (testutil.StubResponse, error){
		"network": func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
			return testutil.StubResponse{}, errors.New("boom")
		},
		"non-2xx": testutil.Sequence(testutil.StubResponse{Status: 404, Body: "{}"}),
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			configure(t)
			install(t, h)
			if SaveUndelivered("x") {
				t.Error("SaveUndelivered = true, want false")
			}
		})
	}
}

func TestSaveUndeliveredUnconfigured(t *testing.T) {
	t.Setenv("GIST_TOKEN", "")
	t.Setenv("GIST_ID", "")
	stub := install(t, testutil.Sequence(testutil.JSON("{}")))
	if SaveUndelivered("x") {
		t.Error("SaveUndelivered = true, want false")
	}
	if len(stub.Requests) != 0 {
		t.Errorf("expected no request when unconfigured, got %d", len(stub.Requests))
	}
}
