/**
 * Business rule constants for DHLottery worker
 *
 * Trace:
 *   spec_id: SPEC-DEPOSIT-001, SPEC-UTILS-001
 *   task_id: TASK-004, TASK-010
 */

/**
 * Minimum deposit amount required for lottery purchase
 * - 5 games × 1,000 KRW = 5,000 KRW (base cost)
 * - 30,000 KRW threshold provides safety buffer
 */
export const MIN_DEPOSIT_AMOUNT = 30000;

/**
 * Charge amount (manual deposit) in KRW
 */
export const CHARGE_AMOUNT = 50000;

/**
 * Lottery purchase game count per run
 */
export const GAMES_PER_PURCHASE = 5;

/**
 * Cost per game in KRW
 */
export const COST_PER_GAME = 1000;

/**
 * Total purchase cost per run in KRW
 */
export const TOTAL_PURCHASE_COST = GAMES_PER_PURCHASE * COST_PER_GAME;
