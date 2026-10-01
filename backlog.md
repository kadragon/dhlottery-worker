# Backlog

## Now

_(empty)_

## Someday

- [ ] [design] Telegram 전송 fallback 채널 — 재시도(3회, 500ms→1500ms, statuses 408/425/429/500/502/503/504) 이미 구현됨. 소진 시 대체 채널 검토. 후보: 보조 `chat_id`, 범용 webhook, Slack.

## Review Backlog

### PR #68 — ledger: retry page fetch once, fail on page-backstop truncation (2026-09-27)

- [ ] [debt] `aggregateWindow`이 `fetched < total`인 상태에서 빈 페이지를 받거나 `data.total`이 없거나 0인데 목록이 비어 있지 않으면 여전히 ok=true를 반환함. 엄격화하기 전에 realtest로 서버의 total 의미를 확인할 것 (source: code-review) — internal/dhlottery/check.go `aggregateWindow` *(blocked by: realtest run — `ledger_total_anomaly` warn 유무 확인)*

### PR #72 — checkpoint: monthly forced rescan, gist rate-limit retry, realtest resume check, shared retry helper (2026-10-01)

- [ ] [debt] `httpclient.Retry`가 결과를 반환하지 않아 호출부 3곳이 외부 변수 캡처로 결과를 전달함(telegram은 attempt 카운터도 별도 유지) — 결과·attempt index를 다루는 generic `Retry[T]` 검토 (source: code-review) — internal/httpclient/retry.go:10
