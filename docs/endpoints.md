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

## Session & Authentication (`src/dhlottery/auth.ts`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/login` | GET | Session init — acquires `DHJSESSIONID` cookie (2026-01: renamed from `JSESSIONID`) |
| `/login/selectRsaModulus.do` | GET | Fetch RSA public key (`rsaModulus`, `publicExponent`) |
| `/login/securityLoginCheck.do` | POST | Submit RSA-encrypted credentials; success = 302 to `loginSuccess.do` or `userId` cookie |

**2026-01 변경:** 평문 POST 로그인 → RSA PKCS#1 v1.5 암호화 (node-forge, DHLottery jsbn.js 호환). 쿠키명 JSESSIONID → DHJSESSIONID. 세션/RSA init 시 301/302 리다이렉트 최대 5회 추적.

## Account Info (`src/dhlottery/account.ts`)

| Endpoint | Method | Purpose | Key field |
|----------|--------|---------|-----------|
| `/mypage/selectUserMndp.do` | GET | Balance (JSON API) | `data.userMndp.crntEntrsAmt` |
| `/lt645/selectThsLt645Info.do` | GET | Current lottery round (JSON API) | `data.result.ltEpsd` |

**2026-01 변경:** `/mypage/home` HTML 파싱 불가(JS 동적 렌더) → `/mypage/selectUserMndp.do` JSON API. `/common.do?method=main` → `/lt645/selectThsLt645Info.do` JSON API.

## Purchase (`src/dhlottery/buy.ts`)

Base: `https://ol.dhlottery.co.kr/olotto/game`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/egovUserReadySocket.json` | POST | Ready socket (pre-purchase handshake) |
| `/execBuy.do` | POST | Execute purchase; requires `saleMdaDcd=10`, `ROUND_DRAW_DATE` (YYYY/MM/DD), `WAMT_PAY_TLMT_END_DT` (추첨일+1년+1일) |

Both require `Origin`, `Referer`, `X-Requested-With` headers.

## Charge Init (`src/dhlottery/charge.ts`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/kbank.do?method=kbankProcess&PayMethod=VBANK&...` | GET | K-Bank 가상계좌 충전 페이지 초기화 (실제 결제 실행 아님, 접근만) |

## Winning Check (`src/dhlottery/check.ts`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/myPage.do?method=lottoBuyList` | GET | 당첨 결과 HTML 파싱. 200 응답만 파싱; 3xx는 진단 로그 후 빈 결과 반환 |

## Pension 720+ Reservation (`src/dhlottery/pension-reserve.ts`)

Base: `https://el.dhlottery.co.kr`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/game/TotalGame.jsp?LottoId=LP72` | GET | Session warm-up (el 도메인 쿠키 획득) |
| `/game/pension720/reserveGame.jsp` | GET | 예약 페이지 (CSRF 토큰, 중복 확인) |
| `/roundRemainTime.do` | GET (AJAX) | 남은 시간 확인 (예약 가능 시간대 검증) |
| `/checkDeposit.do` | POST (AJAX) | 잔액 검증 (예약 전) |
| `/checkMyReserve.do` | POST (AJAX) | 기존 예약 중복 확인 |
| `/addMyReserve.do` | POST (AJAX) | 예약 제출 |

`el.dhlottery.co.kr`는 `JSESSIONID` 쿠키 사용; 없으면 `DHJSESSIONID`로 fallback.

## Telegram (`src/notify/telegram.ts`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `https://api.telegram.org/bot{TELEGRAM_BOT_TOKEN}/sendMessage` | POST | 알림 전송 (전체 플로우 결과를 1회 합산 전송) |
