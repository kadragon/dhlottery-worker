/**
 * Custom Error Classes
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001, SPEC-REFACTOR-P1-ERROR-001
 *   task_id: TASK-002, TASK-REFACTOR-P1-003
 */

/**
 * Base error class for DHLottery operations
 */
export class DHLotteryError extends Error {
  constructor(
    message: string,
    public readonly code: string
  ) {
    super(message);
    this.name = 'DHLotteryError';
  }
}

/**
 * Authentication error
 */
export class AuthenticationError extends DHLotteryError {
  constructor(message: string, code = 'AUTH_ERROR') {
    super(message, code);
    this.name = 'AuthenticationError';
  }
}

/**
 * Network error
 */
export class NetworkError extends DHLotteryError {
  constructor(message: string, code = 'NETWORK_ERROR') {
    super(message, code);
    this.name = 'NetworkError';
  }
}

/**
 * Purchase error
 */
export class PurchaseError extends DHLotteryError {
  constructor(message: string, code = 'PURCHASE_ERROR') {
    super(message, code);
    this.name = 'PurchaseError';
  }
}
