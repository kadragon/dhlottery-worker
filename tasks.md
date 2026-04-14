# Tasks

## Completed

- [x] [doc] Fix misleading auth redirect comment — auth.ts:297 "may not follow" → "is not followed"
- [x] [harness] Add .gitignore entry for .bak files — added `*.bak` to .gitignore
- [x] [debt] Inconsistent logging patterns — verified all modules already use `src/utils/logger.ts`; no direct `console.*` calls in production code

## Stability & Refactoring (2026-04-05 분석)

### Design Decisions Required

- [ ] [design] `checkDeposit` 실패 시 `checkWinning` 스킵 여부 — `src/index.ts:26-37`에서 `checkDeposit` catch가 `return`으로 `checkWinning`까지 건너뜀. 당첨 확인은 구매와 독립적이므로 스킵 불필요할 수 있음. 의도적이라면 주석 추가 필요.
- [ ] [design] `WEEKLY_COMBINED_REQUIRED_BALANCE` 사용 여부 — `src/constants.ts:50`에 정의되어 있으나 `runWorkflow`에서 `TOTAL_PURCHASE_COST`(5000)만 사용. 연금복권 예약 포함 10,000원 체크가 원래 의도인지 확인 필요.
- [ ] [design] Telegram 전송 실패 시 fallback 전략 — `src/notify/telegram.ts:95-109`에서 실패를 로그만 남기고 삼킴. 서비스 핵심 가치가 알림인데 사일런트 실패. 재시도 또는 대체 채널 검토.

### Refactoring

- [x] [debt] 날짜 산술 로직 중복 통합 — `addYearsAndDays`, `addDaysToYmd`를 `src/utils/date.ts`로 이동. buy.ts, pension-reserve.ts에서 import로 교체
- [x] [debt] `reservePensionNextWeek()` 대형 함수 분리 — `verifyDeposit`, `checkForDuplicates`, `submitReservation` 3개 서브함수 추출. 메인 함수 213줄→15줄
- [x] [debt] `auth.ts` 로그인 성공 userId 쿠키 경로 테스트 — TEST-AUTH-005b 추가. 200 응답 + userId 쿠키 조합 테스트
- [x] [debt] `DEBUG` 플래그 환경변수화 — `process.env.DEBUG === 'true'`로 변경. 런타임에 환경변수로 제어 가능

### Dead Code Cleanup

- [x] [debt] `NetworkError` 클래스 제거 — `src/utils/errors.ts:35-40`, 어디서도 throw/catch 안 함. `errors.test.ts`의 관련 테스트도 제거
- [x] [debt] 미사용 타입 정리 — `src/types/auth.types.ts`: `Credentials`, `LoginPayload`, `LoginResponse` 미사용. `src/types/index.ts:50`: `WorkerEnv` 미사용. `src/types/deposit.types.ts`: `DepositEnv` 미사용
- [x] [debt] types 파일의 런타임 값 분리 — `PURCHASE_CONSTANTS` → `src/constants.ts`로 이동. `deposit.types.ts` 파일 삭제 (내용 없음)
- [x] [debt] `check.ts:170` 상태 체크 명확화 — 2xx-only 정책 채택, 3xx는 redirect 진단 메타만 log. 기존 302/500 테스트로 edge case 커버
- [x] [debt] `wrapAuthError` 추출 완료 확인 — 이미 `src/utils/errors.ts:55`에 존재, auth.ts 3개 site에서 사용 중 (과거 리팩터에서 완료됨)
- [x] [debt] `formatKoreanNumber` 통합 완료 확인 — 이미 `src/utils/format.ts` 단일 소스, buy/check/pension-reserve에서 import (과거 리팩터에서 완료됨)

### Architecture Notes

- [ ] [constraint] 쿠키 도메인 격리 없음 — `www`, `ol`, `el.dhlottery.co.kr` 모두 단일 `CookieStore` 사용. 동명 쿠키 덮어쓰기 위험. 구조적 테스트 추가 검토
- [ ] [doc] 에러 처리 패턴 3가지 혼재 문서화 — throw(auth/account) vs return outcome(buy/charge/pension) vs return empty(check). `docs/architecture.md`에 패턴별 사용 기준 문서화
