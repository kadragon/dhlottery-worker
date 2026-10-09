// Package checkpoint persists the lifetime ledger aggregate in a secret GitHub
// gist so each weekly run only queries the ledger since the last settled date.
//
// Every operation is non-fatal: an unconfigured, unreachable, or corrupt gist
// yields nil from Load (the caller falls back to a full ledger scan), and a
// failed Save only logs.
package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
)

// Checkpoint is the settled lifetime aggregate over [Start, Through]
// (dates as YYYY-MM-DD).
type Checkpoint struct {
	Version  int    `json:"version"`
	Start    string `json:"start"`
	Through  string `json:"through"`
	Purchase int    `json:"purchase"`
	Winning  int    `json:"winning"`
}

// schemaVersion identifies the aggregation rules a checkpoint was computed
// under. Bump it whenever those rules change: Load rejects any other version,
// so the next run rescans the full ledger and re-saves under the new rules.
const schemaVersion = 1

// fileName is the gist file holding the checkpoint JSON.
const fileName = "ledger-checkpoint.json"

// undeliveredFileName is the gist file holding the last notification that
// Telegram failed to deliver.
const undeliveredFileName = "undelivered-notification.md"

// gistsURL is the GitHub REST gists endpoint
// (https://docs.github.com/en/rest/gists/gists).
const gistsURL = "https://api.github.com/gists/"

// Injectable seams (overridden in tests).
var (
	doer  httpclient.Doer = &http.Client{Timeout: 30 * time.Second}
	sleep                 = time.Sleep
	now                   = time.Now
)

// retryDelay is the backoff before the single retry of a transient failure.
const retryDelay = 500 * time.Millisecond

// maxRateLimitWait caps how long a rate-limited request waits for its retry;
// a longer server-requested wait skips the retry instead of stalling the run.
const maxRateLimitWait = 10 * time.Second

// resetBuffer is added to a wait derived from x-ratelimit-reset.
const resetBuffer = time.Second

type gistFile struct {
	Content string `json:"content"`
}

type gistPayload struct {
	Files map[string]gistFile `json:"files"`
}

// credentials returns the gist token and id; ok is false when either is unset,
// meaning checkpointing is disabled.
func credentials() (token, id string, ok bool) {
	token, tokenErr := env.Get("GIST_TOKEN")
	id, idErr := env.Get("GIST_ID")
	if tokenErr != nil || idErr != nil {
		return "", "", false
	}
	return token, id, true
}

// requestWithRetry sends the request, retrying once on a transport error, an
// httpclient.TransientStatus, or a GitHub rate limit, so one blip neither
// forces a full rescan (Load) nor drops a checkpoint advance (Save). The retry
// waits retryDelay, or the rate-limit wait GitHub asks for when that is at
// most maxRateLimitWait; a longer one is not retried (the second attempt
// would fail the same way).
func requestWithRetry(method, token, id string, body []byte) (*http.Response, error) {
	var resp *http.Response
	var err error
	httpclient.Retry([]time.Duration{retryDelay}, sleep, func(_ int, final bool) (bool, time.Duration) {
		resp, err = request(method, token, id, body) //nolint:bodyclose // closed here before a retry; the final resp is returned for the caller to close
		var wait time.Duration
		if err == nil {
			var limited bool
			wait, limited = rateLimitWait(resp)
			if !limited && !httpclient.TransientStatus(resp.StatusCode) {
				return false, 0
			}
			if wait > maxRateLimitWait {
				logger.Warn("Ledger checkpoint rate limited beyond retry cap", logger.Fields{
					logger.FieldEvent: "checkpoint_rate_limited", "method": method,
					logger.FieldStatus: resp.StatusCode, "wait": wait.String(),
				})
				return false, 0
			}
		}
		if final {
			return false, 0
		}
		fields := logger.Fields{logger.FieldEvent: "checkpoint_retry", "method": method}
		if err != nil {
			fields[logger.FieldError] = err.Error()
		} else {
			fields[logger.FieldStatus] = resp.StatusCode
			_ = resp.Body.Close()
		}
		logger.Warn("Ledger checkpoint request failed, retrying", fields)
		return true, wait
	})
	return resp, err
}

// rateLimitWait reports whether resp is a GitHub rate-limit response (429, or
// 403 carrying rate-limit headers; a bare 403 is a permission error) and how
// long GitHub asks to wait: Retry-After seconds, else the time until
// x-ratelimit-reset (plus resetBuffer) when x-ratelimit-remaining is 0. A
// zero wait means none was given. An exhausted limit with no parseable reset
// is reported as not rate limited, so a 403 is not retried. See https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api.
func rateLimitWait(resp *http.Response) (time.Duration, bool) {
	if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode != http.StatusForbidden {
		return 0, false
	}
	if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
		return time.Duration(secs) * time.Second, true
	}
	if resp.Header.Get("X-Ratelimit-Remaining") == "0" {
		reset, err := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64)
		if err != nil {
			// Exhausted with no known reset: an early retry cannot succeed.
			return 0, false
		}
		// The buffer absorbs sub-second truncation and runner clock skew.
		return max(time.Unix(reset, 0).Sub(now()), 0) + resetBuffer, true
	}
	return 0, resp.StatusCode == http.StatusTooManyRequests
}

func request(method, token, id string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, gistsURL+id, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "dhlottery-worker") // GitHub asks for an identifying UA
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := doer.Do(req)
	return resp, redact(err)
}

// redact strips the request URL (which embeds the secret gist ID) from a
// transport error before it can reach the logs.
func redact(err error) error {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return fmt.Errorf("%s gist: %w", uerr.Op, uerr.Err)
	}
	return err
}

func loadFailed(err error) {
	logger.Warn("Ledger checkpoint load failed; falling back to full scan", logger.Fields{
		logger.FieldEvent: "checkpoint_load_failed", logger.FieldError: err.Error(),
	})
}

// Load reads the checkpoint from the gist. It returns nil when checkpointing
// is unconfigured or on any fetch/parse error; the caller then does a full scan.
func Load() *Checkpoint {
	token, id, ok := credentials()
	if !ok {
		logger.Info("Ledger checkpoint disabled (GIST_TOKEN/GIST_ID unset)", logger.Fields{
			logger.FieldEvent: "checkpoint_disabled",
		})
		return nil
	}

	resp, err := requestWithRetry(http.MethodGet, token, id, nil)
	if err != nil {
		loadFailed(err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		logger.Warn("Ledger checkpoint load failed; falling back to full scan", logger.Fields{
			logger.FieldEvent: "checkpoint_load_failed", logger.FieldStatus: resp.StatusCode,
		})
		return nil
	}

	var gist gistPayload
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		loadFailed(err)
		return nil
	}
	file, found := gist.Files[fileName]
	if !found {
		loadFailed(errors.New("gist file " + fileName + " not found"))
		return nil
	}
	var cp Checkpoint
	if err := json.Unmarshal([]byte(file.Content), &cp); err != nil {
		loadFailed(err)
		return nil
	}
	if cp.Version != schemaVersion {
		loadFailed(fmt.Errorf("checkpoint schema version %d, want %d", cp.Version, schemaVersion))
		return nil
	}
	return &cp
}

// saveFailed logs a non-fatal save failure and returns false.
func saveFailed(fields logger.Fields) bool {
	fields[logger.FieldEvent] = "checkpoint_save_failed"
	logger.Error("Ledger checkpoint save failed (non-fatal)", fields)
	return false
}

// Save writes cp to the gist. Returns false (after logging) when unconfigured
// or on any error; never aborts the caller.
func Save(cp Checkpoint) bool {
	token, id, ok := credentials()
	if !ok {
		return false
	}

	cp.Version = schemaVersion
	content, err := json.Marshal(cp)
	if err != nil {
		return saveFailed(logger.Fields{logger.FieldError: err.Error()})
	}
	if failure := patchFile(token, id, fileName, string(content)); failure != nil {
		return saveFailed(failure)
	}
	logger.Info("Ledger checkpoint saved", logger.Fields{
		logger.FieldEvent: "checkpoint_saved", "through": cp.Through,
	})
	return true
}

// SaveUndelivered overwrites the gist's undelivered-notification file with
// content, so a notification Telegram failed to deliver is kept somewhere
// private (this repo's run logs are public). Returns false (after logging,
// never the content) when unconfigured or on any error; never aborts the caller.
func SaveUndelivered(content string) bool {
	token, id, ok := credentials()
	if !ok {
		logger.Warn("Undelivered notification not saved (GIST_TOKEN/GIST_ID unset)", logger.Fields{
			logger.FieldEvent: "undelivered_save_skipped",
		})
		return false
	}
	if failure := patchFile(token, id, undeliveredFileName, content); failure != nil {
		failure[logger.FieldEvent] = "undelivered_save_failed"
		logger.Error("Undelivered notification save failed (non-fatal)", failure)
		return false
	}
	logger.Info("Undelivered notification saved to gist", logger.Fields{
		logger.FieldEvent: "undelivered_saved", "file": undeliveredFileName,
	})
	return true
}

// patchFile overwrites one gist file; a PATCH leaves the gist's other files
// intact. It returns the failure as log fields, or nil on success.
func patchFile(token, id, name, content string) logger.Fields {
	body, err := json.Marshal(gistPayload{Files: map[string]gistFile{name: {Content: content}}})
	if err != nil {
		return logger.Fields{logger.FieldError: err.Error()}
	}
	resp, err := requestWithRetry(http.MethodPatch, token, id, body)
	if err != nil {
		return logger.Fields{logger.FieldError: err.Error()}
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return logger.Fields{logger.FieldStatus: resp.StatusCode}
	}
	return nil
}
