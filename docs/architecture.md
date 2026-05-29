# Architecture

## Overview

DHLottery automation: session init → login → check deposit → if sufficient: (pension reserve → lotto purchase) → check winning → Telegram notification.

Stateless. In-memory cookies only. Platform: GitHub Actions scheduled (every Monday 01:00 UTC = KST 10:00).

Core workflow: `init session → login → check deposit → (charge init + warn) or (pension reserve → purchase) → check winning → notify`

Non-critical operations (charge init, pension reserve, winning check, Telegram failures) do not abort the workflow.

## Layer Map

```
cmd/worker/main.go              ← GitHub Actions entry point (run() int)
  └─ internal/workflow          ← Orchestrator (RunWorkflow)
       └─ dhlottery.Client      ← Facade (internal/dhlottery/client.go)
            ├─ auth.go              ← Login (RSA, cookie-based session)
            ├─ account.go           ← Balance/round info
            ├─ charge.go            ← Deposit check & top-up
            ├─ buy.go               ← Lotto purchase (5 games, auto)
            ├─ pension_reserve.go   ← Pension 720+ reservation
            ├─ pension_crypto.go    ← Encryption for pension API
            ├─ check.go             ← Winning results check
            └─ notify.Collector     ← Collects all notifications
                 └─ notify/telegram.go ← Single combined send
```

## Dependency Rules

1. **Domain package** (`internal/dhlottery`) depends on the concrete `*httpclient.Client`, which takes an injectable `Doer` so tests drive it with fakes (no client mock).
2. **Notification** flows one way: domain → `notify.Collector` → `notify.SendCombinedNotification`. Domain code never sends directly.
3. **Types** live with their package — domain types in `internal/dhlottery/types.go`, payloads in `internal/notify`, the HTTP `Response` in `internal/httpclient`.
4. **Leaf packages** (`internal/{env,format,datekst,dherr,logger,constants}`) depend on nothing else in the repo (datekst→none; others→none), keeping the import graph acyclic.

## Module Registry

| Package / file | Responsibility | Key exports |
|----------------|---------------|-------------|
| `cmd/worker/main.go` | Process entry | `run() int`, `main()` |
| `internal/workflow` | Orchestration | `RunWorkflow()` |
| `internal/dhlottery/client.go` | Facade | `Client` |
| `internal/dhlottery/auth.go` | Authentication | `login()` |
| `internal/dhlottery/account.go` | Account info | `getAccountInfo()` |
| `internal/dhlottery/charge.go` | Deposit management | `checkDeposit()` |
| `internal/dhlottery/buy.go` | Lotto purchase | `purchaseLottery()` |
| `internal/dhlottery/pension_reserve.go` | Pension 720+ | `reservePensionNextWeek()` |
| `internal/dhlottery/pension_crypto.go` | Encryption | `EncryptElQ`/`DecryptElQ` |
| `internal/dhlottery/check.go` | Winning check | `checkWinning()` |
| `internal/httpclient` | HTTP client | `New`, `NewWithDoer`, `Client`, `Response` |
| `internal/notify/telegram.go` | Telegram API | `SendCombinedNotification` |
| `internal/notify/collector.go` | Collect payloads | `Collector` |
| `internal/logger` | Logging | `Debug`/`Info`/`Warn`/`Error` (JSON line) |
| `internal/env` | Env boundary | `Get`, `Validate` |
| `internal/dherr` | Error types | `Error`, `New`/`NewAuth`/`NewPurchase`, `WrapAuth` |
| `internal/format` | Formatting | `KoreanNumber`, `Currency` |
| `internal/datekst` | Date helpers | KST date utilities |
| `internal/constants` | Business constants | Amounts, counts, User-Agent |
| `internal/testutil` | Test fake `Doer` | `StubDoer`, `Sequence` |

## Porting Decisions (TS → Go, 2026)

The Go port preserves observable behavior. A few deliberate choices differ from
the original TypeScript and are intentional improvements, validated by an
adversarial TS-vs-Go diff review:

- **RSA**: `crypto/rsa.EncryptPKCS1v15` (hex output) replaces node-forge. Padding
  is randomized, so ciphertext is not byte-comparable; correctness is via
  pubkey-parse + server acceptance.
- **EL `q` URL-decode**: uses `url.PathUnescape` (not `QueryUnescape`) to match
  JS `decodeURIComponent` — a literal `+` in the server's raw base64 is preserved
  rather than turned into a space.
- **Telegram credentials**: read inside the retry loop; a missing/empty token
  yields no outbound request (matching the original throw-and-retry), instead of
  sending a malformed request.
- **Stricter numeric parsing** (improvement): `ltEpsd`/`crntEntrsAmt` decode into
  Go `int`, rejecting non-integer JSON numbers the TS would have accepted. Real
  API values are always integers.
- **Empty-name `Set-Cookie`** (improvement): cookies with an empty name are
  dropped rather than stored and serialized as `=value`.
- **Logging**: `log/slog`-style JSON lines via `internal/logger`; debug gated by
  `DEBUG=true`.

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
