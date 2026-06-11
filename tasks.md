# Tasks

_(no active sprint — deferred work lives in `backlog.md`)_

## Review Backlog

### PR #59 — chore: de-stale planning and quality docs for Go port (2026-06-01)

- [ ] [doc] README.md:17,161 `plan.md` description stale — update to reflect scratch-pad convention (source: pr-review-toolkit:review-pr)

### PR #61 — feat: weekly settlement (2026-06-11)

- [ ] [verify] LP72 연금 누적 구매 단가 불일치: `aggregateLedger`는 모든 행 `prchsQty × CostPerGame(1000)`, 주간 합계는 `PensionReserveCost(5000)` 고정. 픽스처 LP72 행은 `prchsQty:1`. realtest로 실제 원장 LP72 행의 `prchsQty`/단가 확인 후 cumulative 계산을 실측에 맞춰 정정(필요시 `ltGdsCd` 분기). 확정 전 추측 수정 금지. (source: pr-review-toolkit:review-pr, antigravity; plan RISK#1)
- [ ] [enhance] `aggregateLedger` `maxPages(200)` 소진 시 `fetched < total`이면 부분 집계를 정상처럼 반환 — truncation 경고 로그 또는 실패 신호 추가 (source: antigravity, pr-review-toolkit:review-pr)
- [ ] [enhance] `LEDGER_START_DATE` 포맷(YYYYMMDD) 검증 — 무효값이면 `DefaultLedgerStartDate` fallback + 경고 로그. 현재는 무효값이 비200 유발해 결산이 조용히 0/조회실패 처리됨 (source: antigravity)
- [ ] [cleanup] `workflow.go` `RunWorkflow` 말미 `return true` 도달 불가 — settlement always-add로 collector 항상 non-empty (source: pr-review-toolkit:review-pr)
