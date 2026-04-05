@AGENTS.md

## Project Identity

dhlottery-worker: GitHub Actions 기반 자동 복권 구매/알림 서비스 (TypeScript + Bun)

## Toolchain

- **Runtime**: Bun (latest)
- **Language**: TypeScript 6.x, strict mode
- **Test**: Vitest (`bun run test`), coverage thresholds 85%/75%
- **Lint/Format**: Biome (`bun run check:fix`)
- **Typecheck**: `bun run typecheck`
- **Pre-commit**: `bun run precommit` (check:fix + typecheck) via simple-git-hooks

## Conventions

- Single quotes, 2-space indent, semicolons, trailing commas (ES5), LF line endings
- Module pattern: pure functions + facade class (`DHLotteryClient`)
- Types live in `src/types/` with barrel export via `src/types/index.ts`
- Test files colocated: `*.test.ts` next to source
- Notifications collected via `NotificationCollector`, sent as single Telegram message

## Quality Gates (CI)

All must pass before merge:
1. `biome ci .` — lint + format
2. `bun run typecheck` — no type errors
3. `bun run test` — 236 tests green
4. `bun run test:coverage:ci` — coverage thresholds met

## Docs

Knowledge lives in `docs/`. Read on demand.
- `docs/architecture.md` — layer map, module registry
- `docs/runbook.md` — build, test, deploy, common failures
- `docs/quality.md` — quality grades, known gaps
- `docs/eval-criteria.md` — evaluator grading criteria
- `docs/design/{feature}.md` — design decisions
