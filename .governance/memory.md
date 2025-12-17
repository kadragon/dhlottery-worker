# Project Memory

## Snapshot (as of 2025-12-17)

### Purpose
- Cloudflare Workers automation for DHLottery: session → login → account/balance → (optional) deposit init → purchase (5 games) → winning check → Telegram notify.
- Stateless by design: cookies only in-memory during a single execution (no KV / DB).

### Code Layout
- `src/client/http.ts`: HttpClient with automatic cookie capture/attach.
- `src/index.ts`: Scheduled-only entrypoint orchestrating the end-to-end workflow.
- `src/dhlottery/auth.ts`: Login using Secrets (`USER_ID`, `PASSWORD`).
- `src/dhlottery/account.ts`: Parse balance + current round from account HTML.
- `src/dhlottery/charge.ts`: If balance < 30,000 KRW, init deposit page and warn (non-fatal).
- `src/dhlottery/buy.ts`: Two-phase purchase protocol and notify outcome.
- `src/dhlottery/check.ts`: Weekly winning check (previous Mon~Sun KST), notify jackpot only.
- `src/notify/telegram.ts`: Telegram notifier (success/warning/error), never throws.
- Tests: `src/**/*.spec.ts`, fixtures in `src/__fixtures__/`.

### Verified Endpoints / Protocol Notes
- Login: `https://www.dhlottery.co.kr/userSsl.do?method=login`
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

## Next
- (none queued) Consider adding a backlog task to standardize imports via `src/types/index.ts` and reduce console noise in tests.
