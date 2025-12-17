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

## TASK-002: DHLottery Authentication - Completed 2025-12-16

### Overview
Implemented DHLottery authentication module with credential management from Cloudflare Workers Secrets following TDD RED-GREEN-REFACTOR cycle.

### Files Created
- `src/types/auth.types.ts`: Authentication type definitions
- `src/utils/errors.ts`: Custom error classes (DHLotteryError, AuthenticationError, NetworkError)
- `src/dhlottery/auth.ts`: Authentication implementation (95 lines)
- `src/dhlottery/auth.spec.ts`: Test suite with 14 test cases

### Implementation Details
**Authentication Flow:**
- Reads credentials from `env.USER_ID` and `env.PASSWORD` (Cloudflare Workers Secrets)
- Sends POST request to `https://www.dhlottery.co.kr/userSsl.do?method=login`
- Uses `application/x-www-form-urlencoded` content type with URLSearchParams
- Validates response JSON structure and `resultCode` field
- Throws `AuthenticationError` with specific error codes for different failures
- Session cookies automatically managed by HTTP client

**Error Handling:**
- `AUTH_HTTP_ERROR`: Non-200 HTTP status
- `AUTH_UNEXPECTED_RESPONSE`: Invalid JSON or missing fields
- `AUTH_INVALID_CREDENTIALS`: resultCode !== "SUCCESS"
- `AUTH_NETWORK_ERROR`: Network or other errors

**Key Functions:**
- `login(client, env)`: Main authentication function using HTTP client and environment secrets

### Test Coverage
All 4 acceptance tests passed (14 test cases total):
- **TEST-AUTH-001**: Authenticate with valid credentials (3 tests)
- **TEST-AUTH-002**: Reject invalid credentials (4 tests)
- **TEST-AUTH-003**: Use credentials from Secrets (3 tests)
- **TEST-AUTH-004**: Send proper login request format (4 tests)

### Learnings
1. **URLSearchParams Encoding**: Uses `+` for spaces (not `%20`) - standard for form encoding
2. **Error Classification**: Multiple error codes help distinguish failure types
3. **Secrets Management**: Credentials passed via env parameter, never hardcoded
4. **Response Validation**: Always validate response structure before accessing nested properties
5. **Error Wrapping**: Catch and wrap unexpected errors in domain-specific error types

## TASK-003: Account Info Retrieval - Completed 2025-12-16

### Overview
Implemented account information retrieval with HTML parsing to extract deposit balance and current lottery round following TDD RED-GREEN-REFACTOR cycle.

### Files Created
- `src/types/account.types.ts`: AccountInfo interface (balance, currentRound)
- `src/dhlottery/account.ts`: Account info implementation with HTML parsing (121 lines)
- `src/dhlottery/account.spec.ts`: Test suite with 18 test cases
- `src/__fixtures__/account-page.html`: Sample HTML fixture for testing

### Implementation Details
**HTML Parsing Strategy:**
- Regex-based parsing for simple HTML patterns (no full DOM parser needed)
- Balance pattern: `<dd><strong>N,NNN</strong>원</dd>`
- Round pattern: `제<strong>NNNN</strong>회`
- Comma removal before number parsing
- Comprehensive validation of parsed values

**Key Functions:**
- `getAccountInfo(client)`: Main function fetching and parsing account data
- `parseBalance(html)`: Extract and parse deposit balance with comma handling
- `parseRound(html)`: Extract and parse lottery round number
- `validateAccountInfo(balance, round)`: Validate parsed data (balance >= 0, round > 0)

**Error Handling:**
- `ACCOUNT_FETCH_FAILED`: HTTP request failure
- `ACCOUNT_PARSE_BALANCE_FAILED`: Cannot extract or parse balance
- `ACCOUNT_PARSE_ROUND_FAILED`: Cannot extract or parse round number
- `ACCOUNT_INVALID_DATA`: Validation failure (negative balance or invalid round)

### Test Coverage
All 5 acceptance tests passed (18 test cases total):
- **TEST-ACCOUNT-001**: Fetch account info page (3 tests)
- **TEST-ACCOUNT-002**: Parse deposit balance correctly (5 tests)
- **TEST-ACCOUNT-003**: Parse lottery round (3 tests)
- **TEST-ACCOUNT-004**: Validate parsed data (4 tests)
- **TEST-ACCOUNT-005**: Return structured account info (3 tests)

### Learnings
1. **Regex HTML Parsing**: Sufficient for simple patterns; avoided heavy DOM parser dependency
2. **Comma Handling**: Korean number format uses commas (e.g., "45,000") - must remove before parseInt
3. **Fixture-Based Testing**: HTML fixtures enable comprehensive parsing tests without live API calls
4. **Validation Separation**: Separate parsing from validation for clearer error messages
5. **Type Safety**: AccountInfo interface ensures type-safe data usage throughout application
6. **Test Organization**: Group tests by acceptance criteria for clear traceability to specs

## TASK-007: Telegram Notification Service - Completed 2025-12-16

### Overview
Implemented Telegram notification service with support for multiple message types (success, warning, error) and graceful error handling following TDD RED-GREEN-REFACTOR cycle.

### Files Created
- `src/types/notification.types.ts`: Notification type definitions (NotificationType, NotificationPayload, TelegramEnv, TelegramMessage)
- `src/notify/telegram.ts`: Telegram notification implementation (79 lines)
- `src/notify/telegram.spec.ts`: Test suite with 17 test cases

### Implementation Details
**Telegram Bot API Integration:**
- Sends messages to `https://api.telegram.org/bot{token}/sendMessage`
- Uses Markdown parse mode for formatted messages
- Credentials from `env.TELEGRAM_BOT_TOKEN` and `env.TELEGRAM_CHAT_ID` (Cloudflare Workers Secrets)
- Emoji indicators for different notification types (✅ success, ⚠️ warning, ❌ error)
- Graceful error handling - logs errors but never throws

**Message Formatting:**
- Title with emoji prefix based on notification type
- Main message body
- Optional details object formatted as key-value pairs with indentation
- Markdown formatting for readability

**Key Functions:**
- `sendTelegramNotification(env, payload)`: Main notification function with complete error handling
- `formatMessage(payload)`: Format payload into Telegram Markdown message
- `formatDetails(details)`: Format optional details object with proper indentation

**Error Handling:**
- Network errors: Caught and logged, never thrown
- API errors (4xx/5xx): Response parsed and logged with status details
- All failures are non-fatal - service continues execution

### Test Coverage
All 7 acceptance tests passed (17 test cases total):
- **TEST-TELEGRAM-001**: Send formatted messages to Telegram chat (3 tests)
- **TEST-TELEGRAM-002**: Support multiple message types with emoji indicators (3 tests)
- **TEST-TELEGRAM-003**: Format message with title, message, and optional details (3 tests)
- **TEST-TELEGRAM-004**: Use credentials from Secrets (2 tests)
- **TEST-TELEGRAM-005**: Call correct Telegram API endpoint (2 tests)
- **TEST-TELEGRAM-006**: Include parse_mode=Markdown in request (1 test)
- **TEST-TELEGRAM-007**: Handle API failures gracefully without throwing (3 tests)

### Learnings
1. **Graceful Error Handling**: Notification failures should never block main workflow - log and continue
2. **Markdown Formatting**: Telegram supports Markdown for rich text formatting in messages
3. **Emoji Indicators**: Visual indicators improve message scanning in chat history
4. **Optional Details**: Structured details formatting provides flexible debugging information
5. **Secrets Management**: Bot token and chat ID sourced from environment, never hardcoded
6. **Non-Fatal Design**: Notification service is auxiliary - failures must not crash main workflow
7. **Testing Error Scenarios**: Mock fetch rejections and error responses to verify graceful handling

### Integration Points
- Used by TASK-004 (deposit check) for balance warnings
- Used by TASK-005 (lottery purchase) for purchase confirmations
- Used by TASK-006 (winning verification) for win notifications
- Core dependency for all user-facing notifications

## TASK-004: Deposit Check and Charge Initialization - Completed 2025-12-16

### Overview
Implemented deposit balance check and charge page initialization with Chrome MCP verification to discover actual endpoint following TDD RED-GREEN-REFACTOR cycle.

### Files Created
- `src/constants.ts`: Business rule constants (MIN_DEPOSIT_AMOUNT = 30,000 KRW)
- `src/types/deposit.types.ts`: Deposit type definitions (DepositEnv, CHARGE_AMOUNT)
- `src/dhlottery/charge.ts`: Deposit check implementation (91 lines)
- `src/dhlottery/charge.spec.ts`: Test suite with 14 test cases
- `.dev.vars`: Local development environment variables (gitignored)

### Chrome MCP Verification - Critical Discovery
**Pre-Implementation Verification Process:**
1. Used Chrome DevTools Protocol (MCP) to verify actual charge flow
2. Logged into DHLottery with test credentials
3. Navigated through actual charge initialization flow
4. Discovered **actual endpoint differs from specification**

**Spec vs Reality:**
- **Spec mentioned**: `/nicePay.do?method=nicePayInit` (NicePay payment gateway)
- **Actual endpoint**: `/kbank.do?method=kbankProcess` (K-Bank virtual account)
- **Charge amount**: 50,000 KRW (user specified, not in original spec)
- **Virtual account**: Fixed virtual account system with bank transfer
- **Order flow**: GET request to initialize, no POST payment execution

### Implementation Details
**Balance Check Logic:**
- Fetch account info using `getAccountInfo()` from TASK-003
- Compare balance against `MIN_DEPOSIT_AMOUNT` (30,000 KRW)
- Return `true` if sufficient (>= 30,000), `false` if insufficient

**Charge Initialization:**
- URL: `https://www.dhlottery.co.kr/kbank.do?method=kbankProcess`
- Parameters: PayMethod=VBANK, GoodsAmt=50000 (50,000 KRW)
- GET request only (no payment execution)
- Idempotent operation - can be called multiple times

**Notification Integration:**
- **Warning notification** when balance insufficient (type: 'warning')
- **Error notification** when charge initialization fails (type: 'error')
- Includes formatted balance details in Korean number format (N,NNN원)
- Non-fatal - notifications never block execution

**Key Functions:**
- `checkDeposit(client, env)`: Main entry point returning boolean (proceed/block)
- `initializeChargePage(client)`: Access charge page via GET request
- `formatCurrency(amount)`: Format numbers with Korean thousands separator

### Test Coverage
All 5 acceptance tests passed (14 test cases total):
- **TEST-DEPOSIT-001**: Allow purchase when balance sufficient (2 tests)
- **TEST-DEPOSIT-002**: Block purchase when balance insufficient (2 tests)
- **TEST-DEPOSIT-003**: Initialize charge page correctly (3 tests)
- **TEST-DEPOSIT-004**: Notify user when balance low (3 tests)
- **TEST-DEPOSIT-005**: Use correct minimum threshold (3 tests)
- **Error handling**: Account fetch failures (1 test)

### Learnings
1. **Chrome MCP Verification**: Real-world verification prevented incorrect implementation based on outdated spec
2. **Spec Drift**: Website implementations change over time - specs may become outdated
3. **Pre-Implementation Verification**: Testing actual flows before coding saves refactor time
4. **K-Bank Virtual Account**: DHLottery uses fixed virtual account system (not payment gateway)
5. **GET-Only Initialization**: Charge initialization is idempotent GET request, not POST payment
6. **User Input Integration**: Charge amount (50,000 KRW) came from user requirement during verification
7. **Currency Formatting**: Korean number format with thousands separator for user-facing messages
8. **Fail-Safe Design**: Returns false (block purchase) on any errors - prevents accidental purchases
9. **Module Integration**: Successfully integrated with TASK-003 (account) and TASK-007 (telegram)
10. **Environment Variables**: Used .dev.vars for local development credentials (properly gitignored)

### Integration Points
- Depends on TASK-003 (getAccountInfo) for balance retrieval ✓
- Depends on TASK-007 (sendNotification) for user alerts ✓
- Required by TASK-005 (lottery purchase) as pre-check
- Uses constants.ts for MIN_DEPOSIT_AMOUNT threshold

### Chrome MCP Workflow Benefits
- Discovered actual endpoint before writing code
- Verified charge amount (50,000 KRW) through real UI
- Identified virtual account system (not payment gateway)
- Prevented wasted implementation effort on wrong endpoint
- **Lesson**: Always verify external integrations with actual system before coding

## TASK-005: Lottery Purchase Implementation - Completed 2025-12-16

### Overview
Implemented lottery purchase module with automatic number generation for 5 games following TDD RED-GREEN-REFACTOR cycle, with Chrome MCP verification for actual API endpoints.

### Files Created
- `src/types/purchase.types.ts`: Purchase type definitions (85 lines)
- `src/dhlottery/buy.ts`: Purchase implementation (177 lines)
- `src/dhlottery/buy.spec.ts`: Test suite with 18 test cases (776 lines)

### Chrome MCP Verification - API Discovery
**Pre-Implementation Discovery Process:**
1. Used Chrome DevTools Protocol to observe actual purchase workflow
2. Manually executed full purchase flow (auto-generated 5 games)
3. Captured network traffic to discover actual API endpoints
4. Identified two-phase purchase protocol

**Discovered Endpoints:**
- **Phase 1 - Ready**: POST `/olotto/game/egovUserReadySocket.json`
  - Response: `{ direct_yn, ready_ip, ready_time, ready_cnt }`
  - Purpose: Initialize purchase session and get server ID
- **Phase 2 - Execute**: POST `/olotto/game/execBuy.do`
  - Request: URLSearchParams with round, direct (ready_ip), games, amounts, dates
  - Response: `{ loginYn, result: { resultCode, resultMsg } }`
  - Success code: "100"
  - Error codes: "-7" (weekly limit exceeded), others for various failures

**Discovered Error Code:**
- **resultCode "-7"**: Weekly purchase limit exceeded (5,000 KRW/week maximum)
- Message: "[온라인복권 주간 구매한도] 초과되었습니다."

### Implementation Details

**Two-Phase Purchase Protocol:**
1. **preparePurchase()**: Call ready endpoint to initialize session
   - Returns `ready_ip` used in execution phase
   - Validates HTTP 200 response
2. **executePurchase(readyResponse, roundNumber)**: Execute actual purchase
   - Generates 5 game selections (A-E) with auto mode (genType "0")
   - Uses `ready_ip` from phase 1 in `direct` parameter
   - Sends form-encoded request with game parameters

**Game Selection Format:**
```typescript
{
  genType: "0",           // Auto-generated numbers
  arrGameChoiceNum: null, // null for auto mode
  alpabet: "A"|"B"|"C"|"D"|"E"  // Game identifier
}
```

**Purchase Parameters:**
- `round`: Current lottery round number from account info
- `direct`: Server ID from ready endpoint (ready_ip)
- `nBuyAmount`: 5000 (total purchase amount)
- `param`: JSON.stringify(games) - 5 game selections
- `gameCnt`: 5
- `ROUND_DRAW_DATE`: Draw date (7 days from now)
- `WAMT_PAY_TLMT_END_DT`: Payment limit end date (1 year from now)

**Success/Failure Handling:**
- **Success** (resultCode "100"):
  - Return PurchaseSuccess with all purchase details
  - Send Telegram success notification with Korean formatting
  - Include round number, game count, total amount
- **Business Error** (resultCode != "100"):
  - Return PurchaseFailure with error code and message
  - Send Telegram error notification
  - Examples: Weekly limit exceeded, insufficient balance, etc.
- **Network Error**:
  - Catch and wrap in PurchaseFailure
  - Send Telegram error notification
  - Includes error message for debugging

**Key Functions:**
- `purchaseLottery(env)`: Main entry point returning PurchaseOutcome
- `preparePurchase()`: Phase 1 - initialize session
- `executePurchase(readyResponse, roundNumber)`: Phase 2 - execute purchase
- `formatKoreanNumber(amount)`: Format numbers with thousands separator

### Test Coverage
All 5 acceptance test groups passed (18 test cases total):
- **TEST-PURCHASE-001**: Purchase ready endpoint (3 tests)
  - Calls ready endpoint before execution
  - Validates successful response
  - Uses ready_ip in execution phase
- **TEST-PURCHASE-002**: Purchase execution parameters (3 tests)
  - Executes with exactly 5 games
  - Uses automatic number generation (genType "0", arrGameChoiceNum null)
  - Sets total amount to 5,000 KRW
- **TEST-PURCHASE-003**: Parse purchase result (2 tests)
  - Recognizes success with resultCode "100"
  - Includes all purchase details in response
- **TEST-PURCHASE-004**: Telegram success notification (4 tests)
  - Sends success notification on successful purchase
  - Includes game count (5게임)
  - Includes total cost (5,000원)
  - Includes lottery round number (1203회)
- **TEST-PURCHASE-005**: Handle purchase failures (6 tests)
  - Network errors during ready endpoint
  - Network errors during execution
  - Purchase limit exceeded (resultCode "-7")
  - Sends error notifications
  - Atomic transaction (no partial purchases)

### Test Fix Required
**URL Encoding Issue:**
- Initial test failure: Body is URL-encoded, but test checked for literal JSON strings
- **Fix**: Added `decodeURIComponent()` before checking body content
- Reason: execBuy.do uses `application/x-www-form-urlencoded`, so JSON in `param` field is URL-encoded

### Learnings
1. **Two-Phase Protocol**: Many payment/purchase systems use prepare-then-execute pattern
2. **Session Coordination**: ready_ip from phase 1 required in phase 2 (server routing)
3. **Weekly Limits**: DHLottery enforces 5,000 KRW weekly purchase limit per user
4. **Success Code Discovery**: Chrome MCP helped discover resultCode "100" for success
5. **Error Code Mapping**: Negative result codes indicate business rule violations
6. **Form Encoding**: execBuy.do requires URLSearchParams (not JSON body)
7. **Date Formatting**: API expects ISO date format (YYYY-MM-DD) for date fields
8. **Atomic Transactions**: Purchase is all-or-nothing (5 games or nothing)
9. **Korean Formatting**: User-facing messages need Korean number format (5,000원)
10. **Test URL Encoding**: Remember to decode URL-encoded bodies in tests

### Integration Points
- Depends on TASK-003 (getAccountInfo) for round number ✓
- Depends on TASK-007 (sendTelegramNotification) for purchase alerts ✓
- Required by TASK-008 (main orchestration) as core purchase logic
- Uses PURCHASE_CONSTANTS for business rules

### Chrome MCP Workflow Benefits
- Discovered two-phase purchase protocol before coding
- Identified actual endpoint paths and parameter formats
- Captured success code ("100") and error codes ("-7", etc.)
- Verified weekly purchase limit enforcement
- Prevented incorrect implementation based on assumptions
- **Lesson**: Always verify complex workflows with actual system before implementing

### Next Steps
- Begin TASK-006: Winning number verification (new module)
- Then TASK-008: Main orchestration (integrate all modules)
- All dependencies for purchase flow now complete (✓ TASK-001, ✓ TASK-002, ✓ TASK-003, ✓ TASK-004, ✓ TASK-007)
