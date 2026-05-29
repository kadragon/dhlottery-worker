# Conventions

Rules `gofmt`/`go vet` do NOT enforce — don't duplicate what tooling already catches.

## Error Handling

- Use `dherr.Error` (via `dherr.New`/`NewAuth`/`NewPurchase`) for all domain errors. Never return a bare `errors.New`/`fmt.Errorf` from domain modules where a stable `Code` matters.
- Wrap unknown/external errors with `dherr.WrapAuth(err, context)` at authentication boundaries; use `dherr.New` for other domains.
- **Silent swallowing is forbidden.** If a non-critical operation handles an error, it must log it via `internal/logger` and push a notification payload to the collector — never discard silently. (Go's `_ =` discard on a domain error is a smell; see the telegram credential read for the correct pattern.)

### Patterns

Three shapes are in use. Pick the one that matches the orchestrator's treatment of the step:

| Pattern | When to use | Modules |
|---------|------------|---------|
| **Return error** | Downstream work cannot proceed without the value. The orchestrator collects the error into a notification and branches. | `auth.go` (login, initSession, RSA fetch), `account.go` (balance/round fetch) → surfaced via `Client.Login` / `Client.CheckDeposit` |
| **Return outcome** | Failure must not abort the workflow, but the caller wants a typed result. The function handles errors internally, pushes its own notification, then returns a value (never an error). | `buy.go` (`PurchaseOutcome`), `pension_reserve.go` (`PensionReserveOutcome`), `charge.go` (`bool` canPurchase) |
| **Return empty** | Failure and "nothing to report" are indistinguishable to the caller. Notification only fires on non-empty results. | `check.go` — `checkWinning()` returns `nil` on non-fatal HTTP/parse error |

**Anti-patterns:**
- Don't mix patterns in one function (sometimes error, sometimes outcome).
- Don't discard a domain error with `_` to make a step "non-critical" without a deliberate decision and a notification/log.

## HTTP Client

- Domain functions receive `*httpclient.Client` — they never call `net/http` directly.
- The client wraps an injectable `Doer`; tests construct it via `httpclient.NewWithDoer(stub)` so the real cookie/redirect logic is exercised without mocking our own code.
- Redirect following is manual (the real client uses `http.ErrUseLastResponse`). Inspect `Location` headers explicitly (`auth.go fetchWithRedirects`).

## Logging

- All logging uses `internal/logger` (JSON lines). Never call `fmt.Println`/`log.Print` for production logging.
- Debug logs are gated by `DEBUG=true`. Log at the site of the event with enough structured context to diagnose without a debugger.
- Never log secrets or cookie values (see the cookie-overwrite warning, which logs the name only).

## Package Structure

- Domain logic lives in `internal/dhlottery` as unexported functions; the exported facade `Client` assembles them into workflow steps.
- Leaf packages (`env`, `format`, `datekst`, `dherr`, `logger`, `constants`) import nothing else in the repo.
- New domain functions follow the pattern: accept `*httpclient.Client` + typed params → return a typed result or a `dherr.Error`.

## Naming

| Element | Pattern | Example |
|---------|---------|---------|
| Files | `snake_case.go` | `pension_reserve.go` |
| Exported identifiers | `PascalCase` | `EncryptElQ`, `Client`, `KoreanNumber` |
| Unexported identifiers | `camelCase` | `checkDeposit`, `purchaseLottery` |
| Types | `PascalCase` | `AccountInfo`, `PurchaseOutcome` |
| Constants | `PascalCase` (exported) / `camelCase` (pkg) | `TotalPurchaseCost`, `loginURL` |
| Env vars | `SCREAMING_SNAKE_CASE` | `TELEGRAM_BOT_TOKEN` |
| Test files | Same as source + `_test.go` | `auth_test.go` |

## Tests

- Tests are colocated with source (`internal/dhlottery/buy_test.go`).
- Drive the real client through `testutil.StubDoer` — never mock internal functions. Pension tests use real crypto and decrypt requests server-side in the stub.
- Test IDs in comments (e.g., `TEST-AUTH-001`) are informational; don't add new ones unless the suite enforces them.
