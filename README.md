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

1. **Initialize session** → Capture cookies
2. **Authenticate** → Login to DHLottery
3. **Check account** → Fetch balance and lottery info
4. **Verify deposit** → Ensure sufficient balance (≥30,000 KRW)
5. **Purchase lottery** → Buy 5 games (5,000 KRW) if balance OK
6. **Check winning** → Verify results from previous week
7. **Notify** → Send Telegram message with results

## Business Rules

- **Minimum deposit**: 30,000 KRW
- **Purchase amount**: 5 games × 1,000 KRW = 5,000 KRW
- **Purchase mode**: Automatic number generation
- **Winning check**: Previous Monday-Sunday, rank 1 only
- **Execution**: Weekly, Monday 10:00 KST

## Security

- All credentials stored in Cloudflare Workers Secrets
- No sensitive data in logs or code
- Stateless execution (no persistent storage)
- Single-user, personal use only

## Testing

This project uses **Test-Driven Development (TDD)**:

1. Write tests first (RED)
2. Implement minimal code to pass (GREEN)
3. Refactor while keeping tests green (REFACTOR)

All features are specified in `.spec/` directory with Given-When-Then scenarios and acceptance tests.

## Maintenance

### Adding new features

1. Create spec in `.spec/[feature-name]/spec.yaml`
2. Add task to `.tasks/backlog.yaml`
3. Move task to `.tasks/current.yaml`
4. Implement following TDD cycle
5. Update `.governance/memory.md` with learnings

### Debugging

Check Cloudflare Workers logs:
```bash
wrangler tail
```

## License

MIT

## Disclaimer

This is a personal automation tool for individual use only. Not for commercial purposes.
