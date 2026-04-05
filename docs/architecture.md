# Architecture

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
