# Conventions

Rules Biome does NOT enforce — don't duplicate what tooling already catches.

## Error Handling

- Use `DHLotteryError` (or a subclass) for all domain errors. Never throw raw `Error` in domain modules.
- Wrap unknown/external errors with `wrapAuthError(error, context)` at authentication boundaries; use `DHLotteryError` for other domains.
- **Silent swallowing is forbidden.** If a non-critical operation catches an error, it must log it via `logger` and push a notification payload to the collector — never discard silently.

## HTTP Client

- Domain modules receive an `HttpClient` interface — they never call `fetch` directly.
- All HTTP calls go through the client passed into the module function; this is what makes mocking in tests possible.
- Redirect following is manual (no auto-follow). Check `Location` headers explicitly.

## Logging

- All logging uses `src/utils/logger.ts`. Never call `console.log/warn/error` in production code.
- Log at the site of the event, not in callers. Include enough context to diagnose without a debugger.

## Module Structure

- Domain modules export pure functions (not classes). State lives in the caller (`runWorkflow`), not in module scope.
- The facade (`DHLotteryClient`) is the only place where domain functions are assembled into a workflow step.
- New modules must follow the pattern: accept `HttpClient` + typed params → return typed result or throw a domain error.

## Naming

| Element | Pattern | Example |
|---------|---------|---------|
| Files | `kebab-case.ts` | `pension-reserve.ts` |
| Functions | `camelCase` verbs | `checkDeposit()`, `purchaseLottery()` |
| Types/Interfaces | `PascalCase` | `AccountInfo`, `PurchaseResult` |
| Constants | `SCREAMING_SNAKE_CASE` | `TOTAL_PURCHASE_COST` |
| Env vars | `SCREAMING_SNAKE_CASE` | `TELEGRAM_BOT_TOKEN` |
| Test files | Same as source + `.test.ts` | `auth.test.ts` |

## Tests

- Tests are colocated with source (`src/dhlottery/buy.test.ts`).
- Mock at the `HttpClient` boundary — never mock internal module functions.
- Test IDs in comments (e.g., `TEST-AUTH-001`) are informational; don't add new ones unless the suite enforces them.
