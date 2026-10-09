package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Injectable seams (overridden in tests).
var (
	doer    httpDoer = &http.Client{Timeout: 30 * time.Second}
	sleepFn          = time.Sleep
)

// Delay between attempts; len(retryDelays)+1 = total attempts.
var retryDelays = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond}

var (
	mdSpecialRe = regexp.MustCompile("[_*`\\[]")
	upperRe     = regexp.MustCompile(`[A-Z]`)
)

type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// escapeMarkdown escapes Telegram Markdown v1 special characters. Backslash is
// escaped first to avoid double-escaping.
func escapeMarkdown(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	return mdSpecialRe.ReplaceAllString(text, `\$0`)
}

func typeEmoji(t Type) string {
	switch t {
	case Success:
		return "✅"
	case Warning:
		return "⚠️"
	case Error:
		return "❌"
	default:
		return ""
	}
}

// titleCaseKey converts camelCase keys to a spaced, leading-capital form,
// matching the original key formatting. Non-ASCII keys (e.g. Korean) pass
// through unchanged.
func titleCaseKey(key string) string {
	spaced := upperRe.ReplaceAllString(key, " $0")
	runes := []rune(spaced)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return strings.TrimSpace(string(runes))
}

func formatMessage(p Payload) string {
	lines := []string{
		typeEmoji(p.Type) + " **" + escapeMarkdown(p.Title) + "**",
		"",
		escapeMarkdown(p.Message),
	}
	if len(p.Details) > 0 {
		lines = append(lines, "")
		for _, kv := range p.Details {
			lines = append(lines, "- "+escapeMarkdown(titleCaseKey(kv.Key))+": "+escapeMarkdown(kv.Value))
		}
	}
	return strings.Join(lines, "\n")
}

// sendOnce reads credentials and performs a single request. A credential or
// request-build failure is returned as an error (no network call), matching the
// original where getEnv throws inside the retry loop — so missing env never
// produces a malformed outbound request.
func sendOnce(text string) (*http.Response, error) {
	botToken, err := env.Get("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return nil, err
	}
	chatID, err := env.Get("TELEGRAM_CHAT_ID")
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(telegramMessage{ChatID: chatID, Text: text, ParseMode: "Markdown"})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.telegram.org/bot"+botToken+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return doer.Do(req)
}

// sendTelegramMessage posts text to the Telegram API, retrying transient
// failures. Returns true on success, false after exhausting retries or on a
// permanent error.
func sendTelegramMessage(text, failureEvent string) bool {
	maxAttempts := len(retryDelays) + 1
	sent := false

	httpclient.Retry(retryDelays, sleepFn, func(i int, final bool) (bool, time.Duration) {
		resp, err := sendOnce(text)
		if err != nil {
			if !final {
				logger.Warn("Telegram send failed, retrying", logger.Fields{
					logger.FieldEvent: "telegram_retry_attempt", "attempt": i + 1, "error": err.Error(),
				})
				return true, 0
			}
			logger.Error("Failed to send Telegram notification", logger.Fields{
				logger.FieldEvent: failureEvent, "error": err.Error(),
			})
			return false, 0
		}

		status := resp.StatusCode
		_ = resp.Body.Close()

		if status >= 200 && status < 300 {
			sent = true
			return false, 0
		}

		if httpclient.TransientStatus(status) {
			if !final {
				logger.Warn("Telegram API error, retrying", logger.Fields{
					logger.FieldEvent: "telegram_retry_attempt", "attempt": i + 1, "status": status,
				})
				return true, 0
			}
			logger.Error("Telegram notification failed after retries", logger.Fields{
				logger.FieldEvent: "telegram_final_failure", "failureEvent": failureEvent,
				"attempts": maxAttempts, "status": status,
			})
			return false, 0
		}

		// Permanent client error.
		logger.Error("Telegram API error", logger.Fields{
			logger.FieldEvent: "telegram_api_error", "status": status,
		})
		return false, 0
	})
	return sent
}

// SendCombinedNotification formats multiple payloads as a single Telegram
// message separated by "---". Returns false if the send fails after all
// retries; true if sent or empty.
func SendCombinedNotification(payloads []Payload) bool {
	if len(payloads) == 0 {
		return true
	}
	return sendTelegramMessage(FormatCombined(payloads), "telegram_combined_send_failed")
}

// FormatCombined renders payloads as the single combined message text,
// separated by "---".
func FormatCombined(payloads []Payload) string {
	parts := make([]string, len(payloads))
	for i, p := range payloads {
		parts[i] = formatMessage(p)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// SendNotification sends a single notification. Returns false on failure.
func SendNotification(payload Payload) bool {
	return sendTelegramMessage(formatMessage(payload), "telegram_send_failed")
}
