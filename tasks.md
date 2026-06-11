# Tasks

_(no active sprint — deferred work lives in `backlog.md`)_

## Review Backlog

### PR #59 — chore: de-stale planning and quality docs for Go port (2026-06-01)

- [ ] [doc] README.md:17,161 `plan.md` description stale — update to reflect scratch-pad convention (source: pr-review-toolkit:review-pr)

### PR #61 — feat: weekly settlement (2026-06-11)

- [x] [verify] LP72 연금 누적 단가 — **realtest 실측 완료: 버그 아님.** 연금 예약 1건 = 원장 5행 × `prchsQty:1` (ep별 5행). `Σ(prchsQty×1000)=5×1000=5000`/예약 = 정확. 균일식 유지. (resolved fix/ledger-range-chunk)
- [x] [bug] **lifetime 단일 쿼리가 빈 결과 반환** — 서버가 조회 date span 상한(~90~180d) 두고 초과 시 `total=0`(200). realtest ladder로 확인(90d OK, 180d empty, 단 과거 90d 윈도우도 데이터 있음=retention 아닌 span cap). → **90일 청크 walk + 윈도별 페이징**으로 수정. (fix/ledger-range-chunk)
- [x] [bug] `aggregateLedger` `maxWindows` 백스톱 소진 시 `(partial, ok=true)` 반환 — all-or-nothing 위반. → 소진 시 `(zero, false)`+로그. (resolved PR #62 review: antigravity, silent-failure-hunter)
- [x] [bug] 무효 `LEDGER_START_DATE`("foo","2026-6-1")가 `start>end` lexical guard로 `(zero, true)` 묵묵히 통과 → 실제 ₩0 보고. → `time.Parse` 검증, 무효 시 `(zero, false)`. (resolved PR #62 review: codex)
- [ ] [enhance] `aggregateWindow` `maxPages(200)` 소진 시 `fetched < total`이면 부분 집계 무음 반환 — 윈도당 truncation 경고/실패 신호. (윈도≤90d라 발생 가능성 낮음) (source: antigravity, pr-review-toolkit:review-pr)
- [ ] [cleanup] `workflow.go` `RunWorkflow` 말미 `return true` 도달 불가 — settlement always-add로 collector 항상 non-empty (source: pr-review-toolkit:review-pr)
- [ ] [perf] 누적 집계가 매주 ~27 순차 요청(2020→현재, 90d×27). all-or-nothing이라 1건 실패 시 전체 `조회 실패`. 요청 수/신뢰성 트레이드오프 재검토(부분 보고 flag, 캐싱, 또는 시작일 상향). (source: advisor)
