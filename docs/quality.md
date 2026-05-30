# Quality Grades

> ⚠️ Historical snapshot of the pre-2026 **TypeScript** codebase. File names and
> grades below (e.g. `buy.ts`, Vitest counts) predate the Go port and are kept
> for reference only. See `docs/architecture.md` for the current Go structure.

## Coverage (as of init)

- **Tests**: 236 passing across 21 test files
- **Thresholds**: lines 85%, functions 85%, branches 75%, statements 85%
- **All modules** have colocated test files

## Per-Domain Grades

| Domain | Grade | Notes |
|--------|-------|-------|
| Auth (`auth.ts`) | B+ | 26 tests; catch-and-rethrow duplication (see backlog) |
| Purchase (`buy.ts`) | B+ | Well-tested; `formatKoreanNumber` duplicated |
| Charge (`charge.ts`) | B | 18 tests; logging inconsistency |
| Check (`check.ts`) | B | 17 tests; loose status check logic |
| Pension (`pension-reserve.ts`) | B+ | 8 tests; clean |
| Client facade | A- | 18 tests; clean delegation |
| Notifications | A- | 24 tests; collector pattern solid |
| Utils | A | 41 tests; well-isolated |
| Types | A | Centralized, barrel export |

## Known Gaps

1. Auth error wrapping duplication (3+ sites)
2. `formatKoreanNumber` duplicated in buy.ts and check.ts
3. Inconsistent logging strategy across modules
4. Misleading comment in auth.ts about redirect behavior
5. Loose response status check in check.ts
