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

## TASK-001: HTTP Client Implementation - Completed 2025-12-16

### Overview
Implemented HTTP client with automatic cookie management following TDD RED-GREEN-REFACTOR cycle.

### Files Created
- `src/types/http.types.ts`: Type definitions for HTTP client interface
- `src/client/http.ts`: HTTP client implementation with cookie management
- `src/client/http.spec.ts`: Test suite with 11 test cases

### Implementation Details
**Cookie Management Pattern:**
- Used `Headers.getSetCookie()` to capture Set-Cookie headers (returns string array)
- Stored cookies in simple key-value object (`CookieStore`)
- Serialized cookies to `Cookie` header format for subsequent requests
- Automatic cookie updates on each request/response cycle

**Key Functions:**
- `parseCookie()`: Extracts name/value from Set-Cookie header
- `serializeCookies()`: Formats cookie store to "name=value; name=value" format
- `createHttpClient()`: Factory function returning HttpClient instance

### Test Coverage
All 3 acceptance tests passed (11 test cases total):
- **TEST-SESSION-001**: Cookie capture from Set-Cookie headers
- **TEST-SESSION-002**: Cookie inclusion in subsequent requests
- **TEST-SESSION-003**: Cookie update handling

### Learnings
1. **Cloudflare Workers Fetch API**: Uses standard fetch API with `getSetCookie()` method
2. **Cookie Parsing**: Set-Cookie header format: "name=value; attributes..."
3. **Cookie Serialization**: Cookie header format: "name1=value1; name2=value2"
4. **Stateless Design**: No persistent storage, cookies live only during execution
5. **Testing Pattern**: Mock global fetch with vi.fn() for isolated testing

### Next Steps
- Begin TASK-002: DHLottery authentication module
- Follow same TDD cycle
- Reuse HTTP client for authenticated requests
