# Coding Style Guide

## Language
- TypeScript (strict mode enabled)
- Target: ES2021
- Module: ESNext

## Naming Conventions

### Files
- kebab-case for file names: `http-client.ts`, `telegram-notify.ts`
- Suffix patterns:
  - `.ts` for implementation
  - `.spec.ts` for tests
  - `.types.ts` for type definitions

### Variables & Functions
- camelCase: `fetchAccountInfo`, `sendTelegramMessage`
- Boolean variables: `is*`, `has*`, `should*` prefix
- Async functions: clearly indicate async nature in naming

### Types & Interfaces
- PascalCase: `AccountInfo`, `LotteryPurchaseResult`
- Interface prefix: avoid `I` prefix
- Type suffix: use descriptive names without `Type` suffix

### Constants
- UPPER_SNAKE_CASE for true constants: `MIN_DEPOSIT_AMOUNT`, `GAMES_PER_PURCHASE`
- camelCase for configuration objects

## Code Organization

### Imports
1. External libraries
2. Internal modules (absolute paths)
3. Types
4. Constants

```typescript
// External
import { Context } from '@cloudflare/workers-types';

// Internal
import { httpClient } from './client/http';
import { login } from './dhlottery/auth';

// Types
import type { AccountInfo } from './types';

// Constants
import { MIN_DEPOSIT_AMOUNT } from './constants';
```

### Function Structure
- Single responsibility principle
- Max function length: 50 lines (guideline, not hard limit)
- Early returns for error cases
- Explicit return types

### Error Handling
- Use custom error classes
- Always include context in error messages
- No silent failures
- Log errors before rethrowing

## TypeScript Rules

### Strict Mode
- `strict: true`
- No implicit any
- No unused variables/imports
- Explicit function return types for exported functions

### Type Assertions
- Avoid `as` assertions unless absolutely necessary
- Prefer type guards
- Document reason for any type assertion

## Comments

### When to Comment
- Complex business logic
- Non-obvious workarounds
- External API interactions
- Security-sensitive code

### When NOT to Comment
- Self-explanatory code
- Redundant descriptions

### Format
```typescript
// Single-line for brief explanations

/**
 * Multi-line JSDoc for:
 * - Exported functions
 * - Complex types
 * - Public APIs
 */
```

## Async/Await
- Prefer async/await over raw Promises
- Always handle Promise rejections
- Use Promise.all for parallel operations
- Avoid nested async callbacks

## Testing
- Test file naming: `*.spec.ts`
- One test file per implementation file
- Descriptive test names: `should [expected behavior] when [condition]`
- AAA pattern: Arrange, Act, Assert

## Formatting
- Use Prettier (if configured)
- Indent: 2 spaces
- Max line length: 100 characters
- Semicolons: required
- Trailing commas: es5
- Single quotes for strings

## Environment-Specific
### Cloudflare Workers
- Use Fetch API (no Node.js HTTP)
- Respect execution time limits
- Minimize cold start impact
- No file system access
