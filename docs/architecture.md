# Architecture

## Overview

DHLottery automation: session init → login → check deposit → if sufficient: (pension reserve → lotto purchase) → check winning → Telegram notification.

Stateless. In-memory cookies only. Platform: GitHub Actions scheduled (every Monday 01:00 UTC = KST 10:00).

Core workflow: `init session → login → check deposit → (charge init + warn) or (pension reserve → purchase) → check winning → notify`

Non-critical operations (charge init, pension reserve, winning check, Telegram failures) do not abort the workflow.

## Layer Map

```
run.ts                    ← GitHub Actions entry point
  └─ index.ts             ← Orchestrator (runWorkflow)
       └─ DHLotteryClient ← Facade (src/dhlottery/client.ts)
            ├─ auth.ts         ← Login (cookie-based session)
            ├─ account.ts      ← Balance/round info
            ├─ charge.ts       ← Deposit check & top-up
            ├─ buy.ts          ← Lotto purchase (5 games, auto)
            ├─ pension-reserve.ts  ← Pension 720+ reservation
            ├─ pension-crypto.ts   ← Encryption for pension API
            ├─ check.ts        ← Winning results check
            └─ NotificationCollector ← Collects all notifications
                 └─ telegram.ts ← Single combined send
```

## Dependency Rules

1. **Domain modules** (`src/dhlottery/*`) depend on `HttpClient` interface, never on concrete HTTP implementation.
2. **Notification** flows one way: domain → collector → telegram. Domain modules never send directly.
3. **Types** are centralized in `src/types/` — no inline type definitions in domain modules.
4. **Utils** (`src/utils/`) are leaf nodes — they depend on nothing in `src/`.
5. **Constants** (`src/constants.ts`) are pure values — no imports from other src modules.

## Module Registry

| Module | Responsibility | Key exports |
|--------|---------------|-------------|
| `src/index.ts` | Orchestration | `runWorkflow()` |
| `src/run.ts` | Process entry | `main()` |
| `src/dhlottery/client.ts` | Facade | `DHLotteryClient` |
| `src/dhlottery/auth.ts` | Authentication | `login()` |
| `src/dhlottery/account.ts` | Account info | `getAccountInfo()` |
| `src/dhlottery/charge.ts` | Deposit management | `checkDeposit()` |
| `src/dhlottery/buy.ts` | Lotto purchase | `purchaseLottery()` |
| `src/dhlottery/pension-reserve.ts` | Pension 720+ | `reservePensionNextWeek()` |
| `src/dhlottery/pension-crypto.ts` | Encryption | Crypto helpers |
| `src/dhlottery/check.ts` | Winning check | `checkWinning()` |
| `src/client/http.ts` | HTTP client | `createHttpClient()` |
| `src/notify/telegram.ts` | Telegram API | `sendCombinedNotification()` |
| `src/notify/notification-collector.ts` | Collect payloads | `NotificationCollector` |
| `src/utils/logger.ts` | Logging | `logger` |
| `src/utils/env.ts` | Env validation | `validateEnv()` |
| `src/utils/errors.ts` | Error types | Custom error classes |
| `src/utils/format.ts` | Formatting | `formatKoreanNumber()` |
| `src/utils/date.ts` | Date helpers | Date utilities |
| `src/constants.ts` | Business constants | Amounts, counts |

## Business Rules

- **Minimum balance**: 5,000 KRW (5 games × 1,000 KRW). If insufficient: charge init + warning; purchase skipped.
- **Purchase**: 5 auto-pick games, 5,000 KRW total. Atomic — partial purchase not allowed.
- **Winning check**: Previous week (Mon–Sun, KST). 1st-prize results only trigger Telegram alert.
- **Pension reserve**: Next week's round, auto-reserved (el.dhlottery.co.kr).

## Key Decisions

- **RSA login (2026-01)**: Plaintext → RSA PKCS#1 v1.5 encryption (node-forge). Uses `securityLoginCheck.do`.
- **Cookie (2026-01)**: `JSESSIONID` → `DHJSESSIONID`. Login success: 302 Location contains `loginSuccess.do` or `userId` cookie present.
- **Domain (2026-01)**: `dhlottery.co.kr` non-www → `www.dhlottery.co.kr` 301. Session/RSA init follows up to 5 redirects (301/302).
- **Login response mode**: manual redirect. 3xx may indicate success only if Location contains `loginSuccess.do`.
- **Lottery round fetch (2026-01)**: HTML → `/lt645/selectThsLt645Info.do` JSON API (`ltEpsd` field). HTML unparseable (JS-rendered).
- **Balance fetch (2026-01)**: `/mypage/home` HTML → `/mypage/selectUserMndp.do` JSON API (`crntEntrsAmt` field).
- **Purchase request (2026-01)**: `execBuy.do` now requires `saleMdaDcd=10`, `ROUND_DRAW_DATE`, `WAMT_PAY_TLMT_END_DT`, and `Origin`/`Referer`/`X-Requested-With` headers.
- **Winning check redirect**: On 3xx, return empty result without parsing (200-only).
- **el.dhlottery.co.kr cookie**: Uses `JSESSIONID`; falls back to `DHJSESSIONID` if absent.
