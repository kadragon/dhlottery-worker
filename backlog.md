# Backlog

## Now

_(empty)_

## Next

- [ ] [perf] 누적 집계가 매주 ~27 순차 요청(2020→현재, 90d×27). 단발 오류는 PR #68의 페이지 1회 재시도로 흡수됨. 남은 과제: 요청 수·소요 시간 트레이드오프(부분 보고 flag, 캐싱, 또는 시작일 상향). (source: PR #61 review — advisor; narrowed in PR #68 review)

## Someday

- [ ] [design] Telegram 전송 fallback 채널 — 재시도(3회, 500ms→1500ms, statuses 408/425/429/500/502/503/504) 이미 구현됨. 소진 시 대체 채널 검토. 후보: 보조 `chat_id`, 범용 webhook, Slack.

## Review Backlog

### PR #68 — ledger: retry page fetch once, fail on page-backstop truncation (2026-09-27)

- [ ] [debt] `aggregateWindow`이 `fetched < total`인 상태에서 빈 페이지를 받거나 `data.total`이 없거나 0인데 목록이 비어 있지 않으면 여전히 ok=true를 반환함. 엄격화하기 전에 realtest로 서버의 total 의미를 확인할 것 (source: code-review) — internal/dhlottery/check.go:232
- [ ] [debt] ledger 재시도 정책 정리: 일시 오류(network/5xx/408/429)만 재시도, 첫 시도 실패는 Warn으로 기록, telegram `retryStatuses`/`sleepFn`와 공용 helper로 통합 검토 (source: code-review) — internal/dhlottery/check.go:246
