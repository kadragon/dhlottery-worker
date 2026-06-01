# Backlog

## Now

_(empty)_

## Next

_(empty)_

## Someday

- [ ] [design] Telegram 전송 fallback 채널 — 재시도는 이미 구현됨
  (`internal/notify/telegram.go` `sendTelegramMessage`, 3회 시도 500ms→1500ms,
  retryStatuses 408/425/429/5xx). 모든 재시도 소진 시 대체 채널 검토.
  후보: 보조 `chat_id`, 범용 webhook, Slack. 현재 env 경계는
  `TELEGRAM_BOT_TOKEN`/`TELEGRAM_CHAT_ID` 둘뿐 (`internal/env/env.go`).
