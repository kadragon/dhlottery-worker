# Backlog

## Now

_(empty)_

## Review Backlog

### PR #68 — ledger: retry page fetch once, fail on page-backstop truncation (2026-09-27)

- [ ] [debt] `aggregateWindow`이 `fetched < total`인 상태에서 빈 페이지를 받거나 `data.total`이 없거나 0인데 목록이 비어 있지 않으면 여전히 ok=true를 반환함. 엄격화하기 전에 realtest로 서버의 total 의미를 확인할 것 (source: code-review) — internal/dhlottery/check.go `aggregateWindow` *(blocked by: realtest run — `ledger_total_anomaly` warn 유무 확인)*
