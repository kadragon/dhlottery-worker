# External Endpoints

All endpoints verified against current source code (2026-04-28).

## Base Domains

| Domain | Purpose |
|--------|---------|
| `https://www.dhlottery.co.kr` | Main site (session, login, account, purchase, winning) |
| `https://ol.dhlottery.co.kr` | Lotto purchase socket/buy APIs |
| `https://el.dhlottery.co.kr` | Pension 720+ reservation |
| `https://api.telegram.org` | Notification delivery |

Non-www (`dhlottery.co.kr`) redirects to `www` with 301. Always use `www` as base.

## Session & Authentication (`internal/dhlottery/auth.go`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/login` | GET | Session init — acquires `DHJSESSIONID` cookie (2026-01: renamed from `JSESSIONID`) |
| `/login/selectRsaModulus.do` | GET | Fetch RSA public key (`rsaModulus`, `publicExponent`) |
| `/login/securityLoginCheck.do` | POST | Submit RSA-encrypted credentials; success = 302 to `loginSuccess.do` or `userId` cookie |

**2026-01 change:** Plaintext POST login → RSA PKCS#1 v1.5 encryption (Go `crypto/rsa`, compatible with DHLottery jsbn.js). Cookie renamed JSESSIONID → DHJSESSIONID. Session/RSA init follows up to 5 redirects (301/302).

## Account Info (`internal/dhlottery/account.go`)

| Endpoint | Method | Purpose | Key field |
|----------|--------|---------|-----------|
| `/mypage/selectUserMndp.do` | GET | Balance (JSON API) | `data.data.userMndp.crntEntrsAmt` |
| `/lt645/selectThsLt645Info.do` | GET | Current lottery round (JSON API) | `data.data.result.ltEpsd` |

**2026-01 change:** `/mypage/home` HTML unparseable (JS-rendered) → `/mypage/selectUserMndp.do` JSON API. `/common.do?method=main` → `/lt645/selectThsLt645Info.do` JSON API.

## Purchase (`internal/dhlottery/buy.go`)

Base: `https://ol.dhlottery.co.kr/olotto/game`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/egovUserReadySocket.json` | POST | Ready socket (pre-purchase handshake) |
| `/execBuy.do` | POST | Execute purchase; requires `saleMdaDcd=10`, `ROUND_DRAW_DATE` (YYYY/MM/DD), `WAMT_PAY_TLMT_END_DT` (추첨일+1년+1일) |

Both require `Origin`, `Referer`, `X-Requested-With` headers.

## Charge Init (`internal/dhlottery/charge.go`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/kbank.do?method=kbankProcess&PayMethod=VBANK&...` | GET | K-Bank virtual account charge page init (access only — no payment executed) |

## Winning Check (`internal/dhlottery/check.go`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/mypage/selectMyLotteryledger.do` | GET | Purchase/winning ledger (JSON). Covers lotto (`LO40`) and pension (`LP72`). 200 only; 3xx returns empty result after logging. |

Query params: `srchStrDt`/`srchEndDt` (YYYYMMDD, previous-week range), `pageNum=1`, `recordCountPerPage=50`, plus empty `sort`/`ltGdsCd`/`winResult`/`lramSmam`. Headers mirror the RSA-key fetch (`Accept: application/json…`, `X-Requested-With`, `ajax: true`, `Referer: /mypage/mylotteryledger`).

Response: `data.list[]` rows. A win is detected on `ltWnAmt > 0` (`null` = undrawn, `0` = lost). Fields used: `ltEpsd` (round), `ltGdsNm` (product), `ltWnAmt` (prize), `wnRnk` (rank, may be null), `ltWnResult` (label).

**Lifetime settlement (same endpoint, `aggregateLedger`):** the weekly 주간 결산 block recomputes cumulative purchase (Σ `prchsQty` × 1000, covering both LO40 `prchsQty=5` and LP72 `prchsQty=1`×5-rows-per-reserve) and cumulative winning (Σ `ltWnAmt`>0). The API **silently caps a single query's date span** — `srchStrDt..srchEndDt` over ~180 days returns an empty list (200, `total=0`); 90 days works. So the `[LEDGER_START_DATE, today]` range (default `20200101`, optional env) is walked in **contiguous non-overlapping 90-day windows** (newest first, each paged via `data.total`). No state is stored — re-runs recompute identically. Non-fatal & all-or-nothing: any window error returns `ok=false` and the settlement shows 누적/결산 as `조회 실패` rather than a partial total.

**2026-01 change:** the legacy `/myPage.do?method=lottoBuyList` HTML page was retired and now 302-redirects to `/errorPage`; winning detection had been silently returning empty since then. Migrated to this JSON ledger API (mirrors the balance/round HTML→JSON migration).

## Pension 720+ Reservation (`internal/dhlottery/pension_reserve.go`)

Base: `https://el.dhlottery.co.kr`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/game/TotalGame.jsp?LottoId=LP72` | GET | Session warm-up (el 도메인 쿠키 획득) |
| `/game/pension720/reserveGame.jsp` | GET | 예약 페이지 (CSRF 토큰, 중복 확인) |
| `/roundRemainTime.do` | POST (AJAX) | Remaining time check (validates reservation window) |
| `/checkDeposit.do` | POST (AJAX) | Balance check (before reservation) |
| `/checkMyReserve.do` | POST (AJAX) | Duplicate reservation check |
| `/addMyReserve.do` | POST (AJAX) | Submit reservation |

`el.dhlottery.co.kr` uses `JSESSIONID` cookie; falls back to `DHJSESSIONID` if absent.

## Telegram (`internal/notify/telegram.go`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}/sendMessage` | POST | Send notification (all workflow results combined into one message) |
