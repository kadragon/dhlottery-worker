package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func installDoer(t *testing.T, handler func(int, testutil.RecordedRequest) (testutil.StubResponse, error)) *testutil.StubDoer {
	t.Helper()
	t.Setenv("TELEGRAM_BOT_TOKEN", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")
	t.Setenv("TELEGRAM_CHAT_ID", "987654321")

	stub := &testutil.StubDoer{Handler: handler}
	origDoer, origSleep := doer, sleepFn
	doer = stub
	sleepFn = func(time.Duration) {}
	t.Cleanup(func() { doer, sleepFn = origDoer, origSleep })
	return stub
}

func okResp() testutil.StubResponse { return testutil.StubResponse{Status: 200, Body: `{"ok":true}`} }
func statusResp(s int) testutil.StubResponse {
	return testutil.StubResponse{Status: s, Body: `{"ok":false}`}
}

func TestSendNotificationURLAndBody(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(okResp()))

	ok := SendNotification(Payload{Type: Success, Title: "Test Title", Message: "Test Message"})
	if !ok {
		t.Fatal("expected success")
	}
	if len(stub.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(stub.Requests))
	}
	req := stub.Requests[0]
	if !strings.Contains(req.URL, "https://api.telegram.org/bot") ||
		!strings.Contains(req.URL, "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11") ||
		!strings.Contains(req.URL, "/sendMessage") {
		t.Errorf("unexpected URL: %s", req.URL)
	}
	if req.Method != http.MethodPost {
		t.Errorf("method = %s", req.Method)
	}

	var body telegramMessage
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.ChatID != "987654321" {
		t.Errorf("chat_id = %q", body.ChatID)
	}
	if body.ParseMode != "Markdown" {
		t.Errorf("parse_mode = %q", body.ParseMode)
	}
	if !strings.Contains(body.Text, "Test Title") {
		t.Errorf("text missing title: %q", body.Text)
	}
}

func TestFormatMessageDetailsAndEmoji(t *testing.T) {
	cases := []struct {
		typ   Type
		emoji string
	}{
		{Success, "✅"},
		{Warning, "⚠️"},
		{Error, "❌"},
	}
	for _, c := range cases {
		stub := installDoer(t, testutil.Sequence(okResp()))
		SendNotification(Payload{
			Type:    c.typ,
			Title:   "T",
			Message: "M",
			Details: []KV{{Key: "gameCount", Value: "5"}, {Key: "roundNumber", Value: "1145"}},
		})
		var body telegramMessage
		json.Unmarshal([]byte(stub.Requests[0].Body), &body)
		if !strings.Contains(body.Text, c.emoji) {
			t.Errorf("type %s: text missing emoji %s: %q", c.typ, c.emoji, body.Text)
		}
		if !strings.Contains(body.Text, "5") || !strings.Contains(body.Text, "1145") {
			t.Errorf("text missing detail values: %q", body.Text)
		}
		// camelCase keys become title-cased in the rendered output.
		if !strings.Contains(body.Text, "Game Count") || !strings.Contains(body.Text, "Round Number") {
			t.Errorf("keys not title-cased: %q", body.Text)
		}
	}
}

func TestPermanent4xxNoRetry(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(statusResp(400)))
	if SendNotification(Payload{Type: Success, Title: "T", Message: "M"}) {
		t.Error("expected false on 400")
	}
	if len(stub.Requests) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(stub.Requests))
	}
}

func TestPermanent401NoRetry(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(statusResp(401)))
	if SendNotification(Payload{Type: Success, Title: "T", Message: "M"}) {
		t.Error("expected false on 401")
	}
	if len(stub.Requests) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(stub.Requests))
	}
}

func TestNetworkErrorRetriesThenFails(t *testing.T) {
	var errBuf bytes.Buffer
	logger.SetWriters(os.Stdout, &errBuf)
	defer logger.SetWriters(os.Stdout, os.Stderr)

	stub := installDoer(t, func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
		return testutil.StubResponse{}, errors.New("Network error")
	})
	if SendNotification(Payload{Type: Success, Title: "T", Message: "M"}) {
		t.Error("expected false after network failures")
	}
	if len(stub.Requests) != 3 {
		t.Errorf("expected 3 attempts (1+2 retries), got %d", len(stub.Requests))
	}
	if !strings.Contains(errBuf.String(), "telegram_send_failed") {
		t.Errorf("expected failure log, got %q", errBuf.String())
	}
}

func TestRetryOn5xxThenFails(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(statusResp(500)))
	if SendNotification(Payload{Type: Error, Title: "T", Message: "M"}) {
		t.Error("expected false after 5xx")
	}
	if len(stub.Requests) != 3 {
		t.Errorf("expected 3 attempts, got %d", len(stub.Requests))
	}
}

func TestRetryRecoversOn503ThenSuccess(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(statusResp(503), okResp()))
	if !SendNotification(Payload{Type: Success, Title: "T", Message: "M"}) {
		t.Error("expected success after recovery")
	}
	if len(stub.Requests) != 2 {
		t.Errorf("expected 2 attempts, got %d", len(stub.Requests))
	}
}

func TestRetryOn429(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(statusResp(429), okResp()))
	if !SendNotification(Payload{Type: Success, Title: "T", Message: "M"}) {
		t.Error("expected success after 429 retry")
	}
	if len(stub.Requests) != 2 {
		t.Errorf("expected 2 attempts, got %d", len(stub.Requests))
	}
}

func TestSuccessImmediate(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(okResp()))
	if !SendNotification(Payload{Type: Success, Title: "T", Message: "M"}) {
		t.Error("expected immediate success")
	}
	if len(stub.Requests) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(stub.Requests))
	}
}

func TestSendCombined(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(okResp()))
	ok := SendCombinedNotification([]Payload{
		{Type: Success, Title: "First", Message: "First message"},
		{Type: Warning, Title: "Second", Message: "Second message"},
	})
	if !ok {
		t.Error("expected success")
	}
	if len(stub.Requests) != 1 {
		t.Fatalf("expected exactly 1 request, got %d", len(stub.Requests))
	}
	var body telegramMessage
	json.Unmarshal([]byte(stub.Requests[0].Body), &body)
	if !strings.Contains(body.Text, "First") || !strings.Contains(body.Text, "Second") || !strings.Contains(body.Text, "---") {
		t.Errorf("combined text missing parts/divider: %q", body.Text)
	}
}

func TestSendCombinedEmpty(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(okResp()))
	if !SendCombinedNotification(nil) {
		t.Error("empty payloads should return true")
	}
	if len(stub.Requests) != 0 {
		t.Errorf("empty payloads should not call API, got %d requests", len(stub.Requests))
	}
}

func TestSendCombinedFailsAfterRetries(t *testing.T) {
	installDoer(t, func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
		return testutil.StubResponse{}, errors.New("Network error")
	})
	if SendCombinedNotification([]Payload{{Type: Success, Title: "T", Message: "m"}}) {
		t.Error("expected false after all retries fail")
	}
}

func TestMissingEnvDoesNotCallAPI(t *testing.T) {
	stub := installDoer(t, func(int, testutil.RecordedRequest) (testutil.StubResponse, error) {
		t.Error("API must not be called when credentials are missing")
		return testutil.StubResponse{}, nil
	})
	t.Setenv("TELEGRAM_BOT_TOKEN", "") // empty -> env.Get errors

	if SendNotification(Payload{Type: Success, Title: "T", Message: "M"}) {
		t.Error("expected false when token missing")
	}
	if len(stub.Requests) != 0 {
		t.Errorf("no outbound request expected, got %d", len(stub.Requests))
	}
}

func TestEscapeMarkdown(t *testing.T) {
	if got := escapeMarkdown(`a_b*c` + "`d`" + `[e`); !strings.Contains(got, `\_`) || !strings.Contains(got, `\*`) {
		t.Errorf("escapeMarkdown = %q", got)
	}
	if got := escapeMarkdown(`a\b`); !strings.Contains(got, `\\`) {
		t.Errorf("backslash not escaped: %q", got)
	}
}
