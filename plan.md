# Refactoring Plan: Code Quality & Fallback Analysis

## Completed Migrations
- [x] Account balance API migration: `/mypage/home` → `/mypage/selectUserMndp.do` JSON API
- [x] Lottery round API migration: `/common.do?method=main` → `/lt645/selectThsLt645Info.do` JSON API
- [x] RSA authentication implementation (2026-01 update)
- [x] GitHub Actions migration from Cloudflare Workers

## Identified Issues for Refactoring

### HIGH: Auth Error Wrapping Duplication

#### 1. **Catch-and-Rethrow Anti-Pattern (Lines 113-123, 174-181, 321-328)**
Pattern repeats 3+ times:
```typescript
catch (error) {
  if (error instanceof AuthenticationError) {
    throw error;
  }
  const errorMessage = error instanceof Error ? error.message : 'Unknown error';
  throw new AuthenticationError(`Prefix failed: ${errorMessage}`, 'CODE');
}
```

**Why meaningful**:
- Duplicated logic across auth flow, easy to centralize without changing behavior

**Recommendation**: Create `wrapAuthError(error: unknown, context: string): AuthenticationError` utility

---

### HIGH: Misleading Auth Comment (Structural)

#### 2. **Incorrect Redirect Behavior Comment (Lines 262-264)**
Comment says HttpClient follows redirects, but it uses `redirect: 'manual'`.

**Why meaningful**:
- Misleading comment causes confusion and wrong refactor decisions

**Recommendation**: Fix the comment to reflect manual redirect behavior

---

### MEDIUM: Response Status Check Clarity in check.ts

#### 3. **Loose Response Status Check (Line 170)**
```typescript
if (response.status !== 200 && (response.status < 300 || response.status >= 400))
```

Logic is confusing. 3xx typically has empty body, yet we call `.text('euc-kr')`.

**Why meaningful**:
- Improves clarity and avoids relying on empty-body responses

**Recommendation**: Decide explicit policy (2xx only vs. allow 3xx with early return) and encode it clearly

---

### MEDIUM: Inconsistent Error Logging Patterns

#### 4. **Inconsistent Console Output Across Modules**

**Problem**: 
- `auth.ts`: Uses `DEBUG` flag + conditional structured logging (lines 94-104, 159-168, 231-241, etc.)
- `buy.ts`: ALWAYS logs purchase parameters (line 83-93) regardless of DEBUG flag
- `charge.ts`: Uses `console.error()` with JSON in initializeChargePage (lines 41-47, 53-59)
- `check.ts`: Uses `console.error()` in checkWinning (lines 171-177, 207-213)
- `run.ts`: Uses custom structured logging with log() helper
- `telegram.ts`: Uses `console.error()` for API errors (lines 93-101, 108-114)

**Issue**: 
- No consistent logging strategy
- Some modules respect DEBUG flag, others ignore it
- Mix of console.log/console.error/no output
- Some logging should be suppressed in tests but isn't

**Recommendation**: Create centralized Logger with DEBUG flag support

---

### MEDIUM: Redundant Utility Functions

#### 5. **Number Formatting Duplicated Across Modules**

`buy.ts` line 118-120:
```typescript
function formatKoreanNumber(amount: number): string {
  return amount.toLocaleString('ko-KR');
}
```

`check.ts` line 142-144:
```typescript
function formatKoreanNumber(amount: number): string {
  return amount.toLocaleString('ko-KR');
}
```

`charge.ts` line 24-26:
```typescript
function formatCurrency(amount: number): string {
  return `${amount.toLocaleString('ko-KR')}원`;
}
```

**Recommendation**: Move to `src/utils/format.ts` with variants (plain, with currency symbol, etc.)

---

### SMALL: Backup Files Clutter

#### 6. **Leftover .bak Files in src/dhlottery/**

Files: `auth.spec.ts.bak`, `auth.spec.ts.bak2` through `auth.spec.ts.bak7`

**Recommendation**: Add to .gitignore if keeping locally, or delete if no longer needed

---

## Refactoring Priority Order (Tidy First)

1. **P0-STRUCTURAL (No behavior change)**
   - [x] Remove/ignore backup files
   - [x] Fix incorrect redirect behavior comment in auth.ts
   - [x] Extract auth error wrapper utility

2. **P1-STRUCTURAL (Shared utilities/logging)**
   - [x] Consolidate number formatting utilities
   - [x] Create centralized Logger module with DEBUG flag (ensure existing log tests remain valid)

3. **P1-BEHAVIORAL (Requires test change first)**
   - [x] Clarify check.ts status handling policy (2xx only vs 3xx with early return)

## Testing Strategy for Refactoring
- Run existing auth tests after each P1 change (verify no behavior change)
- Run check.ts tests after status code simplification
- Add logging tests for new Logger module
- Verify all tests still pass after utility consolidation
