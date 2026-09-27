# Backlog

## Now

_(empty)_

## Someday

- [ ] [design] Telegram 전송 fallback 채널 — 재시도(3회, 500ms→1500ms, statuses 408/425/429/500/502/503/504) 이미 구현됨. 소진 시 대체 채널 검토. 후보: 보조 `chat_id`, 범용 webhook, Slack.

## Review Backlog

### PR #68 — ledger: retry page fetch once, fail on page-backstop truncation (2026-09-27)

- [ ] [debt] `aggregateWindow`이 `fetched < total`인 상태에서 빈 페이지를 받거나 `data.total`이 없거나 0인데 목록이 비어 있지 않으면 여전히 ok=true를 반환함. 엄격화하기 전에 realtest로 서버의 total 의미를 확인할 것 (source: code-review) — internal/dhlottery/check.go:232
- [ ] [debt] ledger 재시도 정책 정리: 일시 오류(network/5xx/408/429)만 재시도, 첫 시도 실패는 Warn으로 기록, telegram `retryStatuses`/`sleepFn`와 공용 helper로 통합 검토 (source: code-review) — internal/dhlottery/check.go:246

### PR #69 — ledger: incremental lifetime settlement via secret-gist checkpoint (2026-09-27)

- [ ] [debt] 체크포인트에 schema version 필드가 없어 집계 규칙 변경이 기존 누적분에 반영되지 않고, 잘못 저장된 체크포인트를 교정할 주기적 전체 재스캔도 없음 — version 불일치 시 거부 또는 N주마다 전체 재스캔 검토 (source: code-review) — internal/checkpoint/checkpoint.go:44
- [ ] [constraint] `aggregateLedger`(전체 스캔)는 realtest 전용이라 realtest.yml이 운영 경로(settled/tail 분할)를 검증하지 않음 — realtest가 `AggregateLedgerIncremental(…, nil)`을 호출하도록 전환 검토 (source: code-review) — internal/dhlottery/check.go:170
- [ ] [debt] gist Load/Save가 1회 시도뿐이라 일시적 5xx/timeout 한 번에 전체 재스캔 또는 체크포인트 전진 유실 — 위 ledger 재시도 정책 공용 helper와 함께 처리 (source: code-review) — internal/checkpoint/checkpoint.go:86
