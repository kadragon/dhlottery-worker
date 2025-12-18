# DHLottery Worker

Cloudflare Workers-based automated lottery purchase and notification service for DHLottery (Korea Lottery).

## Features

- 🎰 Automatic lottery purchase (5 games per week)
- 💰 Deposit balance monitoring
- 🏆 Winning number verification
- 📱 Telegram notifications
- ⏰ Scheduled execution (every Monday 10:00 KST)

## Architecture

This project follows **Spec-Driven Development (SDD)** and **Test-Driven Development (TDD)** principles:

```
.spec/         → Functional specifications (What)
.tasks/        → Task backlog and tracking (When/What next)
.governance/   → Project memory and patterns (How/Why)
src/           → Implementation code
```

## Project Structure

```
dhlottery-worker/
├── .governance/           # Project memory and coding standards
│   ├── memory.md         # Session memory and learnings
│   ├── coding-style.md   # Code conventions
│   ├── patterns.md       # Design patterns
│   └── env.yaml          # Environment configuration
├── .spec/                # Feature specifications
│   ├── session-management/
│   ├── authentication/
│   ├── account-info/
│   ├── deposit-check/
│   ├── lotto-purchase/
│   ├── winning-check/
│   └── telegram-notification/
├── .tasks/               # Task management
│   ├── backlog.yaml     # Pending tasks
│   ├── current.yaml     # Active task
│   └── done.yaml        # Completed tasks
└── src/                 # Source code
    ├── index.ts         # Entry point
    ├── client/          # HTTP client
    ├── dhlottery/       # DHLottery integration
    ├── notify/          # Notification services
    ├── types/           # Type definitions
    └── utils/           # Utilities
```

## Setup

### Prerequisites

- Node.js 18+
- Cloudflare account
- DHLottery account
- Telegram bot

### Installation

1. Clone repository:
```bash
git clone <repository-url>
cd dhlottery-worker
```

2. Install dependencies:
```bash
npm install
```

3. Configure Cloudflare Workers secrets:
```bash
wrangler secret put USER_ID
wrangler secret put PASSWORD
wrangler secret put TELEGRAM_BOT_TOKEN
wrangler secret put TELEGRAM_CHAT_ID
```

### Development

Run local development server:
```bash
npm run dev
```

Run tests:
```bash
npm test
```

Run tests in watch mode:
```bash
npm run test:watch
```

Type check:
```bash
npm run typecheck
```

### Deployment

Deploy to Cloudflare Workers:
```bash
npm run deploy
```

## Workflow

The service executes automatically every Monday at 10:00 AM KST:

1. **Initialize session** → GET `common.do?method=main` to capture initial cookies
2. **Authenticate** → POST login with session cookies and browser headers
3. **Check account** → Fetch balance from My Page (`myPage.do`) and round from lottery API
4. **Verify deposit** → Ensure sufficient balance (≥5,000 KRW)
5. **Purchase lottery** → Two-phase purchase: ready socket → execute with auto-generated numbers
6. **Check winning** → Verify previous week's results (Mon-Sun KST) for rank 1 jackpots
7. **Notify** → Send Telegram message with purchase and winning results

## Business Rules

- **Minimum deposit**: 5,000 KRW (exact purchase cost)
- **Purchase amount**: 5 games × 1,000 KRW = 5,000 KRW
- **Purchase mode**: Automatic number generation (genType "0")
- **Winning check**: Previous Monday-Sunday (KST), rank 1 (jackpot) only
- **Weekly limit**: DHLottery enforces 5,000 KRW/week per account
- **Execution**: Every Monday 10:00 KST (purchases for upcoming Saturday draw)

## Security

- All credentials stored in Cloudflare Workers Secrets
- No sensitive data in logs or code
- Stateless execution (no persistent storage)
- Single-user, personal use only

## Testing

This project uses **Test-Driven Development (TDD)** and **Spec-Driven Development (SDD)**:

1. Write specification in `.spec/` (Given-When-Then scenarios)
2. Create tests first (RED)
3. Implement minimal code to pass (GREEN)
4. Refactor while keeping tests green (REFACTOR)

### Test Coverage
- **118 tests** covering all modules (auth, account, purchase, winning, notifications, utilities)
- Fixture-based testing with real HTML samples from DHLottery
- Mock HTTP client for deterministic test behavior
- Isolated unit tests per module with spec traceability

Run tests:
```bash
npm test                # Run all tests once
npm run test:watch     # Run tests in watch mode
```

## Maintenance

### Adding new features

1. Create spec in `.spec/[feature-name]/spec.yaml` with GWT scenarios
2. Add task to `.tasks/backlog.yaml` with spec_id reference
3. Move task to `.tasks/current.yaml` 
4. Implement following RED → GREEN → REFACTOR cycle
5. Update `.governance/memory.md` with learnings and patterns
6. Move task to `.tasks/done.yaml` with outcome notes

### Debugging

View Cloudflare Workers logs:
```bash
wrangler tail
```

Check local test execution:
```bash
npm test -- --reporter=verbose
```

Enable debug HTML save (for auth issues):
```bash
export DEBUG_HTML=1
```

## License

MIT

## Implementation Notes

### Authentication Flow (Verified Dec 2025)
- Requires two-phase authentication: session init + login
- GET `common.do?method=main` first to acquire initial cookies
- POST login with browser-like headers (User-Agent, X-Requested-With, Referer, etc.)
- Success indicated by HTML response with `goNextPage` function (NOT JSON)
- Prevents session loss by NOT following redirect after successful login

### Account Info Retrieval
- Fetches both Main Page and My Page to handle potential 302 redirects
- Parses balance from HTML using regex (requires '원' suffix)
- Determines current lottery round via fallback binary search + API call

### Purchase Protocol
- Two-phase atomic transaction: ready socket init → execute with round number
- Auto-generates 5 games using genType "0"
- Returns resultCode "100" on success, "-7" if hitting weekly limit
- Non-fatal: failures logged and notified, do not crash workflow

### Winning Check
- Scans previous week's purchase history (Monday 00:00 ~ Sunday 23:59:59 KST)
- Parses HTML table and filters rank 1 (jackpot) wins only
- Non-fatal: parsing errors return empty, never crash workflow

## Disclaimer

This is a personal automation tool for individual use only. Not for commercial purposes.
