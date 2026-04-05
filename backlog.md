# Backlog

## Auth Error Wrapping Consolidation
> Goal: Eliminate catch-and-rethrow duplication in auth.ts (3+ sites).
> Design: Extract `wrapAuthError(error, context)` utility.
> Done-when: All auth catch blocks use shared helper; no behavior change; all 26 auth tests pass.

- [ ] Extract `wrapAuthError` to `src/utils/errors.ts`
- [ ] Replace catch blocks in `auth.ts` with `wrapAuthError`
- [ ] Verify no behavior change (all tests green)

## Format Utility Consolidation
> Goal: Remove duplicated `formatKoreanNumber` across buy.ts and check.ts.
> Design: Use existing `src/utils/format.ts` as single source.
> Done-when: buy.ts and check.ts import from utils/format; no inline definitions remain.

- [ ] Verify `src/utils/format.ts` has `formatKoreanNumber`
- [ ] Remove inline definitions from buy.ts and check.ts, import from utils
- [ ] Verify all tests pass

## Response Status Check Clarity
> Goal: Simplify confusing status check logic in check.ts line 170.
> Design: Explicit 2xx-only policy or documented 3xx handling.
> Done-when: Status check is readable and intentional; tests cover edge cases.

- [ ] Decide and document status check policy
- [ ] Simplify the conditional
- [ ] Add test for 3xx response behavior
