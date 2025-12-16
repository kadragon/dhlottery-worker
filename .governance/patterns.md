# Design Patterns and Best Practices

## HTTP Client Pattern

### Cookie Management
- Store cookies from `Set-Cookie` headers
- Send cookies in subsequent requests
- Single session per execution (stateless)

```typescript
interface HttpClient {
  request(url: string, options?: RequestInit): Promise<Response>;
  getCookies(): string;
}
```

### Session Lifecycle
1. Initial request → capture cookies
2. Login → update cookies
3. All subsequent requests → include cookies
4. Execution end → discard session

## Error Handling Pattern

### Custom Error Hierarchy
```typescript
class DhlotteryError extends Error {
  constructor(message: string, public readonly context?: Record<string, any>) {
    super(message);
    this.name = 'DhlotteryError';
  }
}

class AuthenticationError extends DhlotteryError {}
class InsufficientBalanceError extends DhlotteryError {}
class PurchaseError extends DhlotteryError {}
```

### Error Propagation
1. Catch errors at function boundary
2. Add context
3. Rethrow or handle
4. Always notify via Telegram for critical errors

## Parsing Pattern

### HTML Parsing
- Use regex for simple extractions
- Use HTMLRewriter for complex DOM operations
- Always validate parsed data
- Provide fallback values

```typescript
function parseBalance(html: string): number {
  const match = html.match(/예치금\s*:\s*([\d,]+)원/);
  if (!match) throw new Error('Balance not found');
  return parseInt(match[1].replace(/,/g, ''), 10);
}
```

## Notification Pattern

### Telegram Message Structure
```typescript
interface NotificationPayload {
  type: 'success' | 'warning' | 'error';
  title: string;
  message: string;
  details?: Record<string, any>;
}
```

### Notification Types
1. **Success**: Purchase completion, winning notification
2. **Warning**: Low balance, charge required
3. **Error**: Authentication failure, system errors

## Testing Patterns

### Mocking External Services
- Mock DHLottery HTTP responses
- Mock Telegram API
- Use fixtures for HTML parsing tests

### Test Data
- Store test HTML fixtures in `__fixtures__/`
- Use realistic data from actual responses
- Sanitize sensitive information

## Workflow Orchestration Pattern

### Main Flow
```typescript
async function main(env: Env): Promise<void> {
  const client = createHttpClient();

  try {
    // 1. Initialize session
    await initSession(client);

    // 2. Authenticate
    await login(client, env.USER_ID, env.PASSWORD);

    // 3. Check account
    const account = await fetchAccountInfo(client);

    // 4. Decide action based on balance
    if (account.balance < MIN_DEPOSIT_AMOUNT) {
      await initChargePage(client);
      await notifyLowBalance(account.balance);
      return;
    }

    // 5. Purchase lottery
    const result = await purchaseLottery(client);
    await notifyPurchaseSuccess(result);

    // 6. Check winning
    const winning = await checkWinning(client);
    if (winning.length > 0) {
      await notifyWinning(winning);
    }
  } catch (error) {
    await notifyError(error);
    throw error;
  }
}
```

### Sequential vs Parallel
- Session/Auth: Sequential (dependencies)
- Purchase + Winning check: Sequential (business logic)
- Multiple notifications: Can be parallel

## Security Patterns

### Secret Management
```typescript
interface Env {
  USER_ID: string;
  PASSWORD: string;
  TELEGRAM_BOT_TOKEN: string;
  TELEGRAM_CHAT_ID: string;
}
```

### Logging Safety
- Never log credentials
- Mask sensitive data in logs
- Use structured logging with context

```typescript
console.log('Purchase completed', {
  games: 5,
  amount: 5000,
  // Never log: userId, password, tokens
});
```

## Retry Pattern

### When to Retry
- Network timeouts
- Temporary server errors (5xx)
- Rate limiting (429)

### When NOT to Retry
- Authentication failures (401)
- Invalid requests (400)
- Resource not found (404)

```typescript
async function withRetry<T>(
  fn: () => Promise<T>,
  maxRetries = 3,
  delay = 1000
): Promise<T> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === maxRetries - 1 || !isRetryable(error)) {
        throw error;
      }
      await sleep(delay * Math.pow(2, i));
    }
  }
  throw new Error('Unreachable');
}
```

## Type Safety Patterns

### Runtime Validation
```typescript
function validateAccountInfo(data: unknown): AccountInfo {
  if (!isObject(data)) throw new Error('Invalid data');
  if (typeof data.balance !== 'number') throw new Error('Invalid balance');
  // ... more validations
  return data as AccountInfo;
}
```

### Branded Types
```typescript
type RoundNumber = number & { __brand: 'RoundNumber' };
type Amount = number & { __brand: 'Amount' };
```
