/**
 * Business rule constants for DHLottery worker
 *
 * Trace:
 *   spec_id: SPEC-DEPOSIT-001, SPEC-UTILS-001, SPEC-REFACTOR-P2-LOG-001
 *   task_id: TASK-004, TASK-010, TASK-REFACTOR-P2-002
 */

/**
 * Debug flag for conditional logging
 * Set to true to enable console.log() output, false to suppress (production)
 */
export const DEBUG = false;

/**
 * Minimum deposit amount required for lottery purchase
 * Same as total purchase cost (no safety buffer)
 * - 5 games × 1,000 KRW = 5,000 KRW
 */
export const MIN_DEPOSIT_AMOUNT = 5000;

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

/**
 * Browser-like User-Agent for requests
 */
export const USER_AGENT =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36';
