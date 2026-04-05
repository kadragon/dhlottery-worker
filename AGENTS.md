# AGENTS.md

## Operational Log

- 동행복권 사이트(dhlottery.co.kr)는 인증/구매/조회 시 EUC-KR 인코딩, 수동 리다이렉트, 쿠키 기반 세션 사용.
- GitHub Actions cron: 매주 월요일 10:00 KST (01:00 UTC). 중복 구매 방지를 위해 워크플로우는 non-throwing 설계.
- Telegram 알림은 단일 메시지로 합쳐서 발송 (NotificationCollector 패턴).
