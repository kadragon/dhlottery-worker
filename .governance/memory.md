# Project Memory

## Project Initialization - 2025-12-16

### Context
- Project: dhlottery-worker
- Purpose: Cloudflare Workers-based automated lottery purchase and notification service
- Target: DHLottery (Korea Lottery) website
- Runtime: Cloudflare Workers (Serverless)
- Language: TypeScript
- Notification: Telegram Bot API

### Architecture Overview
- Stateless serverless architecture
- Cron-based weekly execution (Monday 10:00 KST)
- Single-user personal automation service
- No persistent storage (no KV usage)

### Key Requirements
1. Session management with cookie handling
2. Authentication with DHLottery
3. Account balance verification
4. Automatic lottery purchase (5 games, 5,000 KRW)
5. Winning number verification
6. Telegram notification integration

### Security Constraints
- Credentials stored in Cloudflare Workers Secrets
- No sensitive data in logs or code
- No hardcoded credentials

### Module Structure
```
src/
├─ index.ts              # Entry point with cron trigger
├─ client/http.ts        # HTTP client with cookie management
├─ dhlottery/
│   ├─ auth.ts          # Login and session management
│   ├─ account.ts       # Account info retrieval
│   ├─ buy.ts           # Lottery purchase logic
│   ├─ check.ts         # Winning number verification
│   └─ charge.ts        # Deposit charge page initialization
├─ notify/telegram.ts   # Telegram notification
└─ types/               # TypeScript type definitions
```

### Known Constraints
- Deposit threshold: 30,000 KRW minimum for purchase
- Purchase amount: Fixed 5 games (5,000 KRW)
- No actual payment automation
- Single account only

### Next Steps
- Define all feature specifications
- Create task backlog
- Set up project configuration
- Implement TDD cycle for each module
