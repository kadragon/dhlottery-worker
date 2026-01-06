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
- 마이페이지 (2026-01 변경): `/mypage/home` (구매가능 잔액: divCrntEntrsAmt)
- 구매: `/olotto/game/egovUserReadySocket.json`, `/olotto/game/execBuy.do`
- 충전 init: `/kbank.do?method=kbankProcess` (검증됨)
- 당첨 목록: `/myPage.do?method=lottoBuyList`
- Telegram: `https://api.telegram.org/bot{token}/sendMessage`

## 현재 작업 상태 (.tasks)
- current_task: `null`
- backlog: 비어 있음 (2025-12-20 기준)
- 완료(요약): 세션/로그인/계정/알림/충전/구매/당첨/오케스트레이션/유틸/로그 노이즈/타입 정리 완료
- 리팩터링 P1/P2 및 GitHub Actions 마이그레이션 완료
- 마이그레이션 이후 테스트는 `process.env` mocking 기준으로 갱신 필요(기억 문서 기준)

## 스펙 요약 (.spec)
- SPEC-SESSION-001: 쿠키 기반 세션 수립/갱신 (DHJSESSIONID)
- SPEC-AUTH-001: 인증 흐름(세션 init + 로그인)
- SPEC-AUTH-RSA-001: RSA 암호화 로그인 (2026-01, node-forge 사용)
- SPEC-ACCOUNT-001: 잔액/라운드 파싱
- SPEC-ACCOUNT-002: 새 마이페이지 잔액 파싱 (2026-01, /mypage/home, divCrntEntrsAmt)
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

## 설계/코딩 가이드 핵심 (.governance)
- TypeScript strict, 명확한 이름 규칙, 테스트 파일은 `*.spec.ts`
- 에러는 커스텀 계층 사용, 컨텍스트 포함, 무음 실패 금지
- HTML 파싱은 regex 중심 + 값 검증, 필요한 경우 HTMLRewriter
- 로그에 민감정보 금지, 구조화 로깅 선호

## 핵심 결정/주의사항
- 인증 방식 변경 (2026-01): 평문 → RSA PKCS#1 v1.5 암호화 (node-forge 사용, DHLottery jsbn.js 호환)
- 쿠키 변경: JSESSIONID → DHJSESSIONID, 로그인 성공 시 userId 쿠키 설정
- 로그인 응답은 manual redirect 모드에서 3xx(특히 302)로 성공을 반환할 수 있으나, Location이 `loginSuccess.do`를 포함하는 경우에만 성공으로 처리
- 마이페이지 변경 (2026-01): `/myPage.do` → `/mypage/home`, 잔액 파싱은 `divCrntEntrsAmt` 요소 사용
- 구매/알림/당첨 체크는 실패해도 전체 실행이 중단되지 않도록 설계
- 계정 메인 페이지 fetch 실패 시 단계/URL/Location 정보를 오류 메시지에 포함해 오케스트레이션 가시성 강화
