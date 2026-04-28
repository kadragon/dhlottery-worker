# AGENTS.md

DHLottery-worker: GitHub Actions scheduled job (every Monday 01:00 UTC) that logs in, purchases 5 lotto games, reserves the pension lottery, checks prior week's results, and sends a Telegram notification. TypeScript + Bun, zero external state.

## Docs Index (read on demand)

| File | When to read |
|------|--------------|
| `docs/architecture.md` | Before modifying source structure, adding modules, or touching the orchestrator |
| `docs/endpoints.md` | Before changing HTTP calls, headers, or session flow |
| `docs/conventions.md` | Before writing new modules or error-handling patterns |
| `docs/workflows.md` | When starting any implementation cycle |
| `docs/eval-criteria.md` | When evaluating completed features |
| `docs/runbook.md` | For build, test, deploy commands and troubleshooting |

## Test Runner

- Use `bun run test` (Vitest). **Never** `bun test` — Bun's native runner does not support Vitest's `vi.*` API and fails ~60% of this project's tests.
- Guardrail: `bunfig.toml` preloads `scripts/block-bun-test.ts` which aborts `bun test` when invoked from the repo root. Bun does not walk up for `bunfig.toml`, so running `bun test` from a subdirectory bypasses the guard — always invoke from the repo root.

## Golden Principles

1. **All env reads via `validateEnv()`** — Domain modules never read `process.env.*` directly; `src/utils/env.ts` is the sole boundary.
2. **Non-throwing orchestrator** — Non-critical operations (charge init, pension reserve, winning check, Telegram) must never abort `runWorkflow()`. Errors are collected and reported, not re-thrown to the top level.
3. **Notifications via `NotificationCollector` only** — Domain modules push to the collector; direct calls to `sendCombinedNotification()` are forbidden. One send at the end of the workflow.

## Token Economy

1. Do not re-read a file already read in this session — check the diff/region instead.
2. Do not call tools to confirm information already in context.
3. Run independent tool calls in parallel (reads, grep + glob, etc.).
4. Delegate analysis producing >20 lines of output to a sub-agent; return only the conclusion.
5. Do not restate the user's message.

## Working with Existing Code

- Tests are colocated (e.g., `src/dhlottery/auth.test.ts`). New modules require a test file.
- Coverage thresholds: lines/functions/statements 85%, branches 75%. Run `bun run test:coverage` to verify.
- Pre-commit: `bun run precommit` (Biome check:fix + typecheck). Fix all errors before committing.

## Language Policy

Code, commits, docs: English. User communication: Korean.

## Maintenance

Update this file **only** when ALL of the following are true:

1. Information is not directly discoverable from code / config / manifests / docs
2. It is operationally significant — affects build, test, deploy, or runtime safety
3. It would likely cause mistakes if left undocumented
4. It is stable and not task-specific

**Never add:** architecture summaries, directory overviews, style conventions already enforced by tooling, anything already visible in the repo, or temporary / task-specific instructions.

Prefer modifying or removing outdated entries over appending. When unsure, add a short inline `TODO:` comment rather than inventing guidance.

Size budget: target ≤100 lines, hard warn >200. Move long content to `docs/*.md` and leave a pointer line here.
