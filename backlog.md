# Backlog

## Now

_(empty)_

## Someday

- [ ] [design] Telegram 전송 fallback 채널 — 재시도(3회, 500ms→1500ms, statuses 408/425/429/500/502/503/504) 이미 구현됨. 소진 시 대체 채널 검토. 후보: 보조 `chat_id`, 범용 webhook, Slack.

## Review Backlog

### PR #68 — ledger: retry page fetch once, fail on page-backstop truncation (2026-09-27)

- [ ] [debt] `aggregateWindow`이 `fetched < total`인 상태에서 빈 페이지를 받거나 `data.total`이 없거나 0인데 목록이 비어 있지 않으면 여전히 ok=true를 반환함. 엄격화하기 전에 realtest로 서버의 total 의미를 확인할 것 (source: code-review) — internal/dhlottery/check.go `aggregateWindow` *(blocked by: realtest run — `ledger_total_anomaly` warn 유무 확인)*

### PR #70 — checkpoint: schema version, gist retry, realtest on incremental path (2026-10-01)

- [ ] [debt] 주기적 강제 전체 재스캔 없음 — 첫 전체 스캔에서 한 window가 조용히 total=0을 반환하면 과소 집계된 delta가 체크포인트에 영구 고정됨; N주마다 전체 재스캔 또는 realtest 대조 검토 (source: code-review) — internal/dhlottery/check.go:228
- [ ] [debt] gist 재시도가 고정 500ms로 429 `Retry-After`를 무시하고, GitHub secondary rate limit(403)은 재시도 대상이 아님 — 상한 둔 Retry-After 존중 또는 한계 문서화 (source: code-review) — internal/checkpoint/checkpoint.go:58
- [ ] [constraint] realtest가 prev=nil 전체 스캔만 검증해 resume 경로(validCheckpoint, Through+1 경계, settled/tail 이음)는 실서버로 검증되지 않음 — 읽기 전용 `checkpoint.Load` 후 증분 합계 = 전체 스캔 합계 대조 검토 (source: code-review) — cmd/realtest/main.go:85

### PR #71 — ledger: retry only transient failures, shared TransientStatus, total anomaly diagnostics (2026-10-01)

- [ ] [debt] 재시도 루프가 telegram/checkpoint/ledger 3곳에 각자 sleep seam·지연 상수로 중복 — attempt func + 주입 가능한 sleep을 받는 httpclient 공용 retry helper로 통합 검토 (source: code-review) — internal/checkpoint/checkpoint.go:78
