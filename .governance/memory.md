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

### Project Setup - Completed 2025-12-16

#### Git Repository
- Repository initialized: commit `2162e73`
- Branch: `main`
- Initial commit includes all project structure

#### Structure Created
- `.governance/`: Memory, coding style, patterns, env config (4 files)
- `.spec/`: 7 feature specifications with GWT scenarios
- `.tasks/`: Backlog with 10 tasks, current/done trackers
- Configuration: package.json, tsconfig.json, wrangler.toml, vitest.config.ts
- Documentation: README.md, .gitignore

#### Task Backlog Summary
10 tasks defined in priority order:
1. TASK-001: HTTP client + session management
2. TASK-002: Authentication
3. TASK-003: Account info retrieval
4. TASK-007: Telegram notifications
5. TASK-004: Deposit check
6. TASK-005: Lottery purchase
7. TASK-006: Winning verification
8. TASK-008: Main orchestration
9. TASK-009: Project config (done)
10. TASK-010: Constants and utilities

### Next Steps
- Run `npm install` to install dependencies
- Begin TASK-001: HTTP client implementation
- Follow RED-GREEN-REFACTOR TDD cycle
- Update memory after each major task completion
