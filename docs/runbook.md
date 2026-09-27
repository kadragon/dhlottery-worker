# Runbook

## Prerequisites

- Go 1.26+ (`go.mod` is the single source of truth; workflows read it via `go-version-file: go.mod`)
- Enable the tracked pre-commit hook once per clone: `git config core.hooksPath .githooks` (runs gofmt / vet / test when Go files are staged)

## Commands

| Command | Purpose |
|---------|---------|
| `go build ./...` | Compile all packages |
| `go test ./...` | Run all tests |
| `go test ./... -count=1` | Run tests without the cache |
| `go test ./... -coverprofile=coverage.out` | Run with coverage profile |
| `go tool cover -func=coverage.out` | Per-function coverage report |
| `go tool cover -html=coverage.out` | HTML coverage report |
| `go vet ./...` | Static analysis |
| `gofmt -l ./cmd ./internal` | List files that are not gofmt-clean (empty = clean) |
| `gofmt -w ./cmd ./internal` | Format in place |
| `go run ./cmd/worker` | Run the workflow locally (needs env vars) |
| `DEBUG=true go run ./cmd/worker` | Run with debug-level HTTP logs |
| `go test ./internal/dhlottery/ -run TestLogin -v` | Run a focused test with output |

## Environment Variables

Required in the shell/`.env` (local) or GitHub Secrets (CI):

| Variable | Purpose |
|----------|---------|
| `USER_ID` | 동행복권 로그인 ID |
| `PASSWORD` | 동행복권 로그인 비밀번호 |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API token |
| `TELEGRAM_CHAT_ID` | Telegram chat ID for notifications |
| `DEBUG` | `true` enables debug-level structured logs (optional) |
| `LEDGER_START_DATE` | Lifetime settlement start `YYYYMMDD` (optional, default `20200101`; repo variable) |
| `GIST_TOKEN` | PAT with `gist` scope for the ledger checkpoint (optional; unset → full ledger scan every run) |
| `GIST_ID` | ID of the secret gist holding `ledger-checkpoint.json` (optional, paired with `GIST_TOKEN`) |

Note: there is no built-in `.env` loader. Export the variables in your shell
(e.g. `set -a; source .env; set +a; go run ./cmd/worker`).

## Deployment

- **CI**: Push/PR to `main`/`develop` triggers `.github/workflows/ci.yml`
  (gofmt check → `go vet` → `go test` → coverage gate ≥ 85%).
- **Production**: `.github/workflows/lottery.yml` — cron `0 1 * * 1`
  (every Monday 01:00 UTC = KST 10:00), runs `go run ./cmd/worker`.
- Manual trigger available via `workflow_dispatch`.

## Ledger Checkpoint Setup (one-time, optional)

1. Create a **secret** gist with a file named `ledger-checkpoint.json` (content `{}` is fine — it is rejected as invalid and replaced after the first successful run).
2. Create a PAT with only the `gist` scope.
3. Add repo secrets `GIST_TOKEN` (the PAT) and `GIST_ID` (the gist ID from its URL).

## Exit Codes

`cmd/worker` exits with:

| Code | Meaning |
|------|---------|
| `0` | Workflow ran; notification sent (or nothing to notify) |
| `2` | Workflow ran but the Telegram notification failed after retries |
| `1` | Fatal error before the workflow could complete (e.g. missing env) |

## Common Failures

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Auth failure | Password changed or site maintenance | Update `PASSWORD` secret |
| Purchase fails | Insufficient balance or site down | Check deposit, retry next week |
| Telegram fails | Invalid token or chat ID | Verify secrets |
| 누적 결산 slow / `checkpoint_load_failed` or `checkpoint_disabled` in logs | Gist unset, token expired, or wrong `GIST_ID` | Verify `GIST_TOKEN` (gist scope) / `GIST_ID`; the run still falls back to a full scan |
| Wrong 누적 totals after changing `LEDGER_START_DATE` | — (checkpoint auto-invalidates on start mismatch) | None; to force a rescan, set the gist's `ledger-checkpoint.json` content to `{}` |
| CI coverage gate fails | Total below 85% statement threshold | Add tests for uncovered paths |
| Format check fails | Code not gofmt-clean | Run `gofmt -w ./cmd ./internal` |
| `go vet` fails | Suspicious construct | Fix the reported issue |
