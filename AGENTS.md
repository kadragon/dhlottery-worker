# AGENTS.md

DHLottery-worker: GitHub Actions scheduled job (every Monday 01:00 UTC) that logs in, purchases 5 lotto games, reserves the pension lottery, checks prior week's results, and sends a Telegram notification. Go (1.26+), zero external state. For full pipeline overview, see `docs/architecture.md`.

## Docs Index (read on demand)

| File | When to read |
|------|--------------|
| `docs/architecture.md` | Before modifying source structure, adding modules, or touching the orchestrator |
| `docs/endpoints.md` | Before changing HTTP calls, headers, or session flow |
| `docs/conventions.md` | Before writing new modules or error-handling patterns |
| `docs/workflows.md` | When starting any implementation cycle |
| `docs/eval-criteria.md` | When evaluating completed features |
| `docs/runbook.md` | For build, test, deploy commands and troubleshooting |
| `tasks.md` / `backlog.md` | Sprint contract & deferred work tracking |

## Test Runner

- `go test ./...` runs the suite (no cache: add `-count=1`). Tests are colocated `_test.go` files; integration-style tests drive the real `httpclient.Client` through an injected fake `Doer` (`internal/testutil`), not mocks of our own code.
- Entry point is `func run() int` + `main(){ os.Exit(run()) }` so exit codes are unit-testable.

## Golden Principles

1. **All env reads via `internal/env`** — Domain modules never call `os.Getenv` directly; `internal/env` (`Get`/`Validate`) is the sole boundary.
2. **Non-throwing orchestrator** — Non-critical operations (charge init, pension reserve, winning check, Telegram) must never abort `workflow.RunWorkflow()`. `ReservePensionNextWeek`/`Buy`/`CheckWinning` return outcomes, not errors; only `Login`/`CheckDeposit` return errors, which are collected and reported.
3. **Notifications via `notify.Collector` only** — Domain modules push to the collector via `notify.Notify(payload, collector)`; direct calls to `notify.SendCombinedNotification()` from domain code are forbidden. One send at the end of the workflow.

## Token Economy

1. Do not re-read a file already read in this session — check the diff/region instead.
2. Do not call tools to confirm information already in context.
3. Run independent tool calls in parallel (reads, grep + glob, etc.).
4. Delegate analysis producing >20 lines of output to a sub-agent; return only the conclusion.
5. Do not restate the user's message.

## Working with Existing Code

- Tests are colocated (e.g., `internal/dhlottery/auth_test.go`). New packages/modules require a test file.
- Coverage gate: total statement coverage ≥ 85% (`go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`). CI enforces it.
- Before committing: `gofmt -w ./cmd ./internal && go vet ./... && go test ./...` must all pass.

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
