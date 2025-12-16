/**
 * Deposit Check Types
 *
 * Trace:
 *   spec_id: SPEC-DEPOSIT-001
 *   task_id: TASK-004
 */

/**
 * Environment variables required for deposit check
 */
export interface DepositEnv {
  TELEGRAM_BOT_TOKEN: string;
  TELEGRAM_CHAT_ID: string;
}

/**
 * Charge amount in KRW
 */
export const CHARGE_AMOUNT = 50000;
