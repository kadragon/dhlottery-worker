# Tasks

## Completed

- [x] [doc] Fix misleading auth redirect comment — auth.ts:297 "may not follow" → "is not followed"
- [x] [harness] Add .gitignore entry for .bak files — added `*.bak` to .gitignore
- [x] [debt] Inconsistent logging patterns — verified all modules already use `src/utils/logger.ts`; no direct `console.*` calls in production code

## Stability & Refactoring (2026-04-05 분석)

### Design Decisions Required

- [x] [design] `checkDeposit` 실패 시 `checkWinning` 스킵 여부 — 버그로 판정. `checkDeposit` 실패 시 `canPurchase = false`로 설정 후 `checkWinning` 실행하도록 수정 (PR fix/orchestrator-winning-and-balance-gate).
- [x] [design] `WEEKLY_COMBINED_REQUIRED_BALANCE` 사용 여부 — 게이트로 채택. `runWorkflow`에서 `TOTAL_PURCHASE_COST(5000)` → `WEEKLY_COMBINED_REQUIRED_BALANCE(10000)` 교체 (PR fix/orchestrator-winning-and-balance-gate).
- [x] [design] Telegram 전송 실패 시 fallback 전략 — 3-attempt retry (500ms→1500ms backoff), `sendCombinedNotification` boolean 반환, `run.ts` exit(2). PR #45.

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

- [x] [constraint] 쿠키 도메인 격리 없음 — 공유는 의도된 동작. 동명 덮어쓰기 감지 warning 로그 추가 (`event: cookie_overwritten`, 이름만 로깅). PR #46.
- [x] [doc] 에러 처리 패턴 3가지 혼재 문서화 — throw(auth/account) vs return outcome(buy/charge/pension) vs return empty(check). `docs/conventions.md`에 패턴별 사용 기준 문서화 (PR docs/error-handling-patterns)

## Review Backlog

### PR #41 — scaffold full agent harness: principles, hooks, conventions, and workflows (2026-04-28)

- [x] [doc] `docs/workflows.md` references `tasks.md` but file is not in AGENTS.md Docs Index — added `tasks.md / backlog.md` row to Docs Index (PR docs/pr-41-review-followup)
- [x] [doc] `docs/architecture.md` Overview duplicates AGENTS.md opening paragraph — made `docs/architecture.md` canonical; added pointer line to AGENTS.md opening (PR docs/pr-41-review-followup)
- [x] [doc] `AGENTS.md:19` bun test guardrail — bolded "always invoke from the repo root" in Test Runner section (PR docs/pr-41-review-followup)
