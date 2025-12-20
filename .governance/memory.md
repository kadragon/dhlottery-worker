# Project Memory

## Snapshot (as of 2025-12-17)

### Purpose
- Cloudflare Workers automation for DHLottery: session → login → account/balance → (optional) deposit init → purchase (5 games) → winning check → Telegram notify.
- Stateless by design: cookies only in-memory during a single execution (no KV / DB).

### Code Layout
- `src/client/http.ts`: HttpClient with automatic cookie capture/attach.
- `src/index.ts`: Scheduled-only entrypoint orchestrating the end-to-end workflow.
- `src/dhlottery/auth.ts`: Login using Secrets (`USER_ID`, `PASSWORD`).
- `src/dhlottery/account.ts`: Parse balance from HTML, get current round via public API (getLottoNumber endpoint).
- `src/dhlottery/charge.ts`: If balance < 5,000 KRW (exact purchase cost), init deposit page and warn (non-fatal).
- `src/dhlottery/buy.ts`: Two-phase purchase protocol and notify outcome.
- `src/dhlottery/check.ts`: Weekly winning check (previous Mon~Sun KST), notify jackpot only.
- `src/notify/telegram.ts`: Telegram notifier (success/warning/error), never throws.
- Tests: `src/**/*.spec.ts`, fixtures in `src/__fixtures__/`.

### Verified Endpoints / Protocol Notes
- Session Init: `https://dhlottery.co.kr/common.do?method=main` (GET)
  - Must be called first to acquire initial session cookies before login
  - Returns HTML page with Set-Cookie headers
- Login: `https://www.dhlottery.co.kr/userSsl.do?method=login` (POST)
  - Requires initial session cookies from session init
  - Headers: Content-Type, User-Agent, X-Requested-With, Referer, Sec-Fetch-Site, Connection, sec-ch-ua, sec-ch-ua-mobile
  - Form params: returnUrl, userId, password, checkSave, newsEventYn
  - Response: JSON with `result.resultCode` ("SUCCESS" or "FAIL")
- Account page: `https://www.dhlottery.co.kr/myPage.do?method=myPage`
  - Unauthenticated access redirects to login.
- Deposit init (verified): `https://www.dhlottery.co.kr/kbank.do?method=kbankProcess`
  - GET-only init (no payment automation).
- Purchase (verified):
  - Ready: `https://ol.dhlottery.co.kr/olotto/game/egovUserReadySocket.json` (POST)
  - Execute: `https://ol.dhlottery.co.kr/olotto/game/execBuy.do` (POST, x-www-form-urlencoded)
  - Weekly limit code observed: `-7`
- Winning list: `https://www.dhlottery.co.kr/myPage.do?method=lottoBuyList`
  - Params used: `searchStartDate`, `searchEndDate` (strings), `nowPage` for pagination.

### Patterns / Guardrails
- HTML parsing: prefer targeted regex/TD extraction; validate parsed numbers.
- Encoding: some pages are EUC-KR; avoid parsing that depends on Korean keywords when possible (extract from stable numeric positions).
- Non-fatal auxiliaries: Telegram notify, deposit init, and winning check should not crash the whole run.

## Completed Tasks (high level)
- TASK-001 (SPEC-SESSION-001): HttpClient cookie store + tests.
- TASK-002 (SPEC-AUTH-001): Login module + error types + tests.
- TASK-003 (SPEC-ACCOUNT-001): Account balance/round parsing + fixture + tests.
- TASK-007 (SPEC-TELEGRAM-001): Telegram notifier + formatting + tests (graceful failures).
- TASK-004 (SPEC-DEPOSIT-001): Deposit threshold + K-Bank deposit init + notifications + tests.
- TASK-005 (SPEC-PURCHASE-001): Two-phase purchase + notifications + tests.
- TASK-006 (SPEC-WINNING-001): Previous-week range in KST, parse win list HTML, notify rank==1 only + tests.
- TASK-008 (SPEC-ORCH-001): Scheduled-only Worker entrypoint + orchestration + tests.
- TASK-010 (SPEC-UTILS-001): Shared constants + KST date utilities + types barrel + tests.
- TASK-011 (SPEC-QOL-IMPORTS-001): Standardized type-only imports to use the `src/types` barrel.
- TASK-012 (SPEC-LOGGING-001): Silenced console output during Vitest runs (no runtime behavior changes).
- TASK-013 (SPEC-AUTH-001): Fixed authentication protocol to match DHLottery's requirements (session init + browser headers + form params).

## Recent Improvements (2025-12-17 PM)
- **HttpClient consistency**: Fixed `preparePurchase()` and `executePurchase()` to use HttpClient instead of global fetch (ensures session cookies are properly sent).
- **Date calculation**: Added `getNextSaturdayKst()` utility for accurate lottery draw date calculation (replaces hardcoded Date.now() + 7 days).
- **Notification standardization**: All notification titles now use English (Korean remains in message body for user-facing content).
- **Environment validation**: Added `validateEnv()` function to check required environment variables at Worker startup.
- **Structured logging**: Migrated all `console.log/error` calls to JSON format for better observability in Cloudflare Workers environment.
- **Test coverage**: Added 5 new tests (getNextSaturdayKst tests, env validation test); total: 113 tests passing.

## Recent Improvements (2025-12-18)
- **Authentication protocol fix (TASK-013)**: Fixed "Failed to parse login response" error by implementing proper authentication flow based on n8n workflow analysis:
  - Added `initSession()` function to acquire initial session cookies before login (GET to common.do?method=main)
  - Added browser-like headers to login request (User-Agent, X-Requested-With, Referer, Sec-Fetch-Site, Connection, sec-ch-ua, sec-ch-ua-mobile)
  - Added missing form parameters (returnUrl, checkSave, newsEventYn) to login request
  - Fixed response handling: DHLottery returns HTML on successful login (check for `goNextPage` function), JSON on failure
  - Prevented UID cookie clearing by NOT following redirect after successful login
  - Updated SPEC-AUTH-001 to document two-phase authentication flow
  - Updated all 16 auth tests to verify two-phase flow; all tests passing

- **Account info improvements**: Fixed account balance parsing and round detection:
  - Updated balance regex patterns to match actual HTML structure: `<td class="ta_right">N,NNN 원</td>`
  - Changed round detection from HTML parsing to API-based approach using `common.do?method=getLottoNumber&drwNo=N`
  - Implemented binary search with fallback to estimated round calculation (year - 2002) × 52
  - Updated all 18 account tests to mock both account page HTML and lottery API responses; all tests passing

- **Balance threshold adjustment (TASK-014)**: Reduced minimum deposit requirement from 30,000 KRW to 5,000 KRW:
  - Changed `MIN_DEPOSIT_AMOUNT` constant from 30000 to 5000 (matches exact purchase cost: 5 games × 1,000 KRW)
  - Removed safety buffer - system now allows purchase with exact amount needed
  - Updated all charge tests (14 tests) and constants tests to use new threshold
  - Updated notification messages to reflect new minimum requirement
  - All 115 tests passing

- **Purchase flow hardening (TASK-015)**: Addressed potential request rejection by standardizing `User-Agent` header:
  - Created shared `USER_AGENT` constant in `src/constants.ts` (Chrome/91.0...).
  - Applied `User-Agent` to all outbound requests: `auth.ts`, `buy.ts` (critical for purchase), `account.ts`, `charge.ts`, `check.ts`.
  - Refactored `auth.spec.ts` to remove unused test variables.
  - Updated charge.spec.ts to verify User-Agent presence.
  - Ensured consistency with reference n8n workflow.

- **Account retrieval robustness (TASK-016)**: Modified `getAccountInfo` to prevent `HTTP 302` redirects:
  - Strategy change: Fetch Main Page (`common.do`) first to get round and stabilize session (mimicking browser redirect).
  - Then Fetch My Page (`myPage.do`) for balance as requested.
  - Added fallback: If My Page fails (302/Error), parse balance from Main Page header.
  - Made balance regex stricter (requires '원' suffix) to avoid false positives.
  - Updated tests to verify dual-fetch behavior and fallback logic.

## Verified Workflow Behavior
- **Schedule**: Runs every Monday 10:00 KST (cron: "0 1 * * 1")
  - Monday execution purchases lottery for **upcoming Saturday** (5 days ahead)
  - Monday execution checks **previous week's** winning results (Mon-Sun)
- **Purchase flow**: login → checkDeposit → (if balance sufficient) purchaseLottery → checkWinning
- **Non-fatal components**: Telegram notifications, charge initialization, and winning checks never crash the main workflow.

## Refactoring Plan (2025-12-20)

Based on comprehensive code analysis (REFACTOR_ANALYSIS.md), initiated Phase 1+2 refactoring:

### Phase 1: Quick Wins (~30 minutes)
- **TASK-REFACTOR-P1-001**: Remove non-functional debug HTML write code (auth.ts:171-184)
- **TASK-REFACTOR-P1-002**: Fix type safety for NotificationPayload.details (Record<string, any> → strict types)
- **TASK-REFACTOR-P1-003**: Add PurchaseError class for consistent error handling
- **TASK-REFACTOR-P1-004**: Document unused DHLotteryClient.getAccountInfo() method

### Phase 2: Refactoring (~1 hour)
- **TASK-REFACTOR-P2-001**: Extract regex patterns to named constants (BALANCE_PATTERNS, WINNING_PATTERNS)
- **TASK-REFACTOR-P2-002**: Consolidate logging under conditional DEBUG flag
- **TASK-REFACTOR-P2-003**: Update buy.ts error handling to use PurchaseError
- **TASK-REFACTOR-P2-004**: Review and document session initialization URL strategy

### Analysis Summary
- **Overall Grade**: A- (Excellent)
- **Security**: No vulnerabilities detected
- **Code Quality**: 0 linting errors, 118 tests passing
- **Architecture**: Clean separation of concerns, well-structured
- **Issues**: 7 minor/moderate issues identified, all non-critical
- **Improvements**: Focus on maintainability, consistency, and reducing production log noise

All specs created under `.spec/refactor-phase1-*` and `.spec/refactor-phase2-*`.
Tasks queued in `.tasks/backlog.yaml` ready for execution.

## Next
- Execute TASK-REFACTOR-P1-001 to start Phase 1 refactoring
- Follow TDD approach: update tests first, then implement changes
