# Architecture

## Overview

DHLottery 자동화: 세션 수립 → 로그인 → 잔액/회차 조회 → 잔액 부족 시 충전 init + 경고 → 구매(5게임) → 연금복권 예약 → 당첨 확인 → Telegram 알림.

상태 저장 없음. 실행 단위 메모리 쿠키만 사용. 플랫폼: GitHub Actions 스케줄 실행 (매주 월요일 01:00 UTC = KST 10:00).

핵심 워크플로우: `init session → login → fetch account → check deposit → (charge init + warn) or purchase → pension reserve → check winning → notify`

비치명 작업(충전 init, 연금 예약, 당첨 확인, Telegram 실패)은 전체 플로우를 중단하지 않음.

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

- **최소 잔액**: 5,000 KRW (5게임 × 1,000원). 부족 시 충전 init + 경고 알림 후 구매 스킵.
- **구매**: 자동 번호 5게임, 총 5,000 KRW. 원자적 처리 — 부분 구매 금지.
- **당첨 확인**: 이전 주(월~일, KST 기준). 1등만 Telegram 알림.
- **연금복권 예약**: 다음 주 회차 자동 예약 (el.dhlottery.co.kr).

## Key Decisions

- **RSA 로그인 (2026-01)**: 평문 → RSA PKCS#1 v1.5 암호화 (node-forge). `securityLoginCheck.do` 사용.
- **쿠키 (2026-01)**: `JSESSIONID` → `DHJSESSIONID`. 로그인 성공 판별은 302 Location에 `loginSuccess.do` 포함 여부 또는 `userId` 쿠키 존재 여부.
- **도메인 (2026-01)**: `dhlottery.co.kr` non-www → `www.dhlottery.co.kr` 301. 세션/RSA init에서 301/302 최대 5회 추적.
- **로그인 응답 모드**: manual redirect. 3xx 응답이 성공일 수 있으나 Location이 `loginSuccess.do`를 포함해야만 성공 처리.
- **로또 회차 조회 (2026-01)**: HTML → `/lt645/selectThsLt645Info.do` JSON API (`ltEpsd` 필드). HTML은 JS 동적 로드로 파싱 불가.
- **잔액 조회 (2026-01)**: `/mypage/home` HTML → `/mypage/selectUserMndp.do` JSON API (`crntEntrsAmt` 필드).
- **구매 요청 보강 (2026-01)**: `execBuy.do`에 `saleMdaDcd=10`, `ROUND_DRAW_DATE`, `WAMT_PAY_TLMT_END_DT` 추가 필수. `Origin`/`Referer`/`X-Requested-With` 헤더 필수.
- **당첨 확인 3xx**: 3xx 리다이렉트 시 HTML 파싱하지 않고 빈 결과 반환 (200 전용).
- **el.dhlottery.co.kr 쿠키**: `JSESSIONID` 사용. 없으면 `DHJSESSIONID` fallback.
