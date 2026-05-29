# Runbook

## Prerequisites

- Go 1.26+

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

Note: there is no built-in `.env` loader. Export the variables in your shell
(e.g. `set -a; source .env; set +a; go run ./cmd/worker`).

## Deployment

- **CI**: Push/PR to `main`/`develop` triggers `.github/workflows/ci.yml`
  (gofmt check → `go vet` → `go test` → coverage gate ≥ 85%).
- **Production**: `.github/workflows/lottery.yml` — cron `0 1 * * 1`
  (every Monday 01:00 UTC = KST 10:00), runs `go run ./cmd/worker`.
- Manual trigger available via `workflow_dispatch`.

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
| CI coverage gate fails | Total below 85% statement threshold | Add tests for uncovered paths |
| Format check fails | Code not gofmt-clean | Run `gofmt -w ./cmd ./internal` |
| `go vet` fails | Suspicious construct | Fix the reported issue |
