package checkpoint

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func install(t *testing.T, h func(int, testutil.RecordedRequest) (testutil.StubResponse, error)) *testutil.StubDoer {
	t.Helper()
	stub := &testutil.StubDoer{Handler: h}
	orig := doer
	doer = stub
	t.Cleanup(func() { doer = orig })
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
		`{"start":"2020-01-01","through":"2026-08-23","purchase":1000,"winning":500}`))))

	cp := Load()
	want := Checkpoint{Start: "2020-01-01", Through: "2026-08-23", Purchase: 1000, Winning: 500}
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
	var got Checkpoint
	if err := json.Unmarshal([]byte(body.Files[fileName].Content), &got); err != nil || got != cp {
		t.Errorf("saved content = %+v (err %v), want %+v", got, err, cp)
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
