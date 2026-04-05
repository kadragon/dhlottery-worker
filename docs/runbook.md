# Runbook

## Prerequisites

- Bun (latest)
- Node.js types (`@types/node`)

## Commands

| Command | Purpose |
|---------|---------|
| `bun install` | Install dependencies |
| `bun run test` | Run all tests |
| `bun run test:watch` | Watch mode |
| `bun run test:coverage` | Coverage report |
| `bun run typecheck` | TypeScript check |
| `bun run check:fix` | Biome lint + format (auto-fix) |
| `bun run precommit` | Pre-commit hook (check:fix + typecheck) |
| `bun run start` | Run workflow locally (needs .env) |

## Environment Variables

Required in `.env` (local) or GitHub Secrets (CI):

| Variable | Purpose |
|----------|---------|
| `USER_ID` | 동행복권 로그인 ID |
| `PASSWORD` | 동행복권 로그인 비밀번호 |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API token |
| `TELEGRAM_CHAT_ID` | Telegram chat ID for notifications |

## Deployment

- **CI**: Push/PR to `main`/`develop` triggers `.github/workflows/ci.yml`
- **Production**: `.github/workflows/lottery.yml` runs every Monday 10:00 KST via cron
- Manual trigger available via `workflow_dispatch`

## Common Failures

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Auth failure | Password changed or site maintenance | Update `PASSWORD` secret |
| Purchase fails | Insufficient balance or site down | Check deposit, retry next week |
| Telegram fails | Invalid token or chat ID | Verify secrets |
| CI coverage gate fails | New code below 85% line threshold | Add tests for uncovered paths |
| Pre-commit fails | Biome or typecheck errors | Run `bun run check:fix` then `bun run typecheck` |
