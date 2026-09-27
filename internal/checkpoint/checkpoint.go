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
	"time"

	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/httpclient"
	"github.com/kadragon/dhlottery-worker/internal/logger"
)

// Checkpoint is the settled lifetime aggregate over [Start, Through]
// (dates as YYYY-MM-DD).
type Checkpoint struct {
	Start    string `json:"start"`
	Through  string `json:"through"`
	Purchase int    `json:"purchase"`
	Winning  int    `json:"winning"`
}

// fileName is the gist file holding the checkpoint JSON.
const fileName = "ledger-checkpoint.json"

// gistsURL is the GitHub REST gists endpoint
// (https://docs.github.com/en/rest/gists/gists).
const gistsURL = "https://api.github.com/gists/"

// doer is the injectable seam (overridden in tests).
var doer httpclient.Doer = &http.Client{Timeout: 30 * time.Second}

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

func request(method, token, id string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, gistsURL+id, body)
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

	resp, err := request(http.MethodGet, token, id, nil)
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

	content, err := json.Marshal(cp)
	if err != nil {
		return saveFailed(logger.Fields{logger.FieldError: err.Error()})
	}
	body, err := json.Marshal(gistPayload{Files: map[string]gistFile{fileName: {Content: string(content)}}})
	if err != nil {
		return saveFailed(logger.Fields{logger.FieldError: err.Error()})
	}

	resp, err := request(http.MethodPatch, token, id, bytes.NewReader(body))
	if err != nil {
		return saveFailed(logger.Fields{logger.FieldError: err.Error()})
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return saveFailed(logger.Fields{logger.FieldStatus: resp.StatusCode})
	}
	logger.Info("Ledger checkpoint saved", logger.Fields{
		logger.FieldEvent: "checkpoint_saved", "through": cp.Through,
	})
	return true
}
