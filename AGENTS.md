# AGENTS.md

## 목적/개요

- DHLottery 자동화: 세션 → 로그인 → 잔액/라운드 조회 → (잔액 부족 시 충전 유도) → 구매(5게임) → 당첨 확인 → Telegram 알림
- 상태 저장 없음(실행 단위 메모리 쿠키만 사용)
- 플랫폼: GitHub Actions 스케줄 실행 (Cloudflare Workers에서 마이그레이션 완료)

## 핵심 워크플로우

- init session → login → fetch account → check deposit → (charge init + warn) or purchase → check winning → notify
- 비치명 작업(충전 init, 당첨 확인, Telegram 실패)은 전체 플로우를 중단하지 않음

## 운영 환경/스케줄

- Runtime: TypeScript (ES2021, ESNext), 실행: `tsx`
- 스케줄: 매주 월요일 01:00 UTC (KST 10:00)
- Secrets: `USER_ID`, `PASSWORD`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` (GitHub Secrets, `process.env`)

## 비즈니스 규칙 (핵심만)

- 최소 잔액: 5,000 KRW (5게임 × 1,000)
- 구매: 자동 번호 5게임, 총 5,000 KRW, 원자적 처리(부분 구매 금지)
- 당첨 확인: 이전 주(월~일, KST), 랭크 1만 알림

## 외부 엔드포인트 요약

- Base: `https://dhlottery.co.kr`
- Session/Login (2026-01 변경):
  - 세션 init: `/login` (DHJSESSIONID 쿠키 발급)
  - RSA 키 조회: `/login/selectRsaModulus.do` (modulus + exponent)
  - 로그인: `/login/securityLoginCheck.do` (RSA 암호화된 credentials POST)
- 로또 회차 정보 (2026-01 변경): `/lt645/selectThsLt645Info.do` (JSON API, ltEpsd 필드)
- 잔액 조회 (2026-01 변경): `/mypage/selectUserMndp.do` (JSON API, crntEntrsAmt 필드)
- 구매: `/olotto/game/egovUserReadySocket.json`, `/olotto/game/execBuy.do`
- 충전 init: `/kbank.do?method=kbankProcess` (검증됨)
- 당첨 목록: `/myPage.do?method=lottoBuyList`
- Telegram: `https://api.telegram.org/bot{token}/sendMessage`

## 스펙 요약 (.spec)

- SPEC-SESSION-001: 쿠키 기반 세션 수립/갱신 (DHJSESSIONID)
- SPEC-AUTH-001: 인증 흐름(세션 init + 로그인)
- SPEC-AUTH-RSA-001: RSA 암호화 로그인 (2026-01, node-forge 사용)
- SPEC-ACCOUNT-001: 잔액/라운드 파싱
- SPEC-ACCOUNT-002: 잔액 파싱 (2026-01, /mypage/selectUserMndp.do JSON API, crntEntrsAmt)
- SPEC-ACCOUNT-003: 로또 회차 파싱 (2026-01, /lt645/selectThsLt645Info.do JSON API, ltEpsd)
- SPEC-DEPOSIT-001: 최소 잔액 체크 + 충전 init + 경고 알림
- SPEC-PURCHASE-001: 5게임 자동 구매 + 결과 알림
- SPEC-WINNING-001: 이전 주 당첨 확인(랭크 1만)
- SPEC-TELEGRAM-001: Telegram 알림 포맷/전송
- SPEC-ORCH-001: 전체 오케스트레이션 흐름
- SPEC-UTILS-001: 공통 상수/날짜 유틸
- SPEC-QOL-IMPORTS-001: 타입 import 표준화
- SPEC-LOGGING-001: 테스트 콘솔 출력 억제
- SPEC-ARCH-001: DHLotteryClient 클래스로 구조화
- SPEC-GHACTION-001: GitHub Actions 마이그레이션

## 핵심 결정/주의사항

- 인증 방식 변경 (2026-01): 평문 → RSA PKCS#1 v1.5 암호화 (node-forge 사용, DHLottery jsbn.js 호환)
- 쿠키 변경: JSESSIONID → DHJSESSIONID, 로그인 성공 시 userId 쿠키 설정
- 로그인 응답은 manual redirect 모드에서 3xx(특히 302)로 성공을 반환할 수 있으나, Location이 `loginSuccess.do`를 포함하는 경우에만 성공으로 처리
- 로또 회차 조회 변경 (2026-01): `/common.do?method=main` → `/lt645/selectThsLt645Info.do` JSON API (HTML은 JS로 동적 로드되어 사용 불가, ltEpsd 필드가 예정 회차 직접 제공)
- 잔액 조회 변경 (2026-01): `/mypage/home`은 JS 동적 렌더링으로 HTML 파싱 불가 → `/mypage/selectUserMndp.do` JSON API 사용 (crntEntrsAmt 필드)
- 구매 요청 보강 (2026-01): `execBuy.do`는 `saleMdaDcd=10`, `ROUND_DRAW_DATE`(YYYY/MM/DD), `WAMT_PAY_TLMT_END_DT`(추첨일+1년+1일) 포함 필요, ready/execBuy는 Origin/Referer/X-Requested-With 헤더 필요
- 구매/알림/당첨 체크는 실패해도 전체 실행이 중단되지 않도록 설계
- 계정 정보 fetch 실패 시 단계/URL/Location 정보를 오류 메시지에 포함해 오케스트레이션 가시성 강화

## 전략적 인사이트/거버넌스 업데이트

- 로깅은 `src/utils/logger.ts`를 사용하고 debug는 `DEBUG`가 true일 때만 출력한다 (info/error는 항상 출력).
- KRW 숫자 포맷은 `src/utils/format.ts` 유틸을 사용한다 (중복 포맷 함수 금지).
- 인증 흐름 예외는 `wrapAuthError`로 감싸 공통 메시지/코드 정책을 유지한다.
- `checkWinning`은 3xx 리다이렉트를 파싱하지 않고 빈 결과로 종료한다 (200만 HTML 파싱).
- `src/dhlottery` 내 `.bak*` 파일은 금지하며 테스트로 감시한다.
- 테스트 파일 네이밍은 `*.test.ts`를 표준으로 사용하며 Vitest include/exclude와 Biome 설정이 이에 맞춰져 있다.
