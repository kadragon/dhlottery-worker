# Backlog

## Now

_(empty)_

## Next

- [ ] [enhance] `aggregateWindow` `maxPages(200)` 소진 시 `fetched < total`이면 부분 집계 무음 반환 — 윈도당 truncation 경고/실패 신호. (윈도≤90d라 발생 가능성 낮음) (source: PR #61 review — antigravity, pr-review-toolkit:review-pr)
- [ ] [cleanup] `workflow.go` `RunWorkflow` 말미 `return true` 도달 불가 — settlement always-add로 collector 항상 non-empty (source: PR #61 review — pr-review-toolkit:review-pr)
- [ ] [perf] 누적 집계가 매주 ~27 순차 요청(2020→현재, 90d×27). all-or-nothing이라 1건 실패 시 전체 `조회 실패`. 요청 수/신뢰성 트레이드오프 재검토(부분 보고 flag, 캐싱, 또는 시작일 상향). (source: PR #61 review — advisor)

## Someday

- [ ] [design] Telegram 전송 fallback 채널 — 재시도(3회, 500ms→1500ms, statuses 408/425/429/500/502/503/504) 이미 구현됨. 소진 시 대체 채널 검토. 후보: 보조 `chat_id`, 범용 webhook, Slack.
