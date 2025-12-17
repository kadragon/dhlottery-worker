/**
 * Deposit Check Types
 *
 * Trace:
 *   spec_id: SPEC-DEPOSIT-001
 *   task_id: TASK-004, TASK-010
 */

import { CHARGE_AMOUNT } from '../constants';

/**
 * Environment variables required for deposit check
 */
export interface DepositEnv {
  TELEGRAM_BOT_TOKEN: string;
  TELEGRAM_CHAT_ID: string;
}

export { CHARGE_AMOUNT };
