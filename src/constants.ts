/**
 * Business rule constants for DHLottery worker
 * Trace: SPEC-DEPOSIT-001, TASK-004
 */

/**
 * Minimum deposit amount required for lottery purchase
 * - 5 games × 1,000 KRW = 5,000 KRW (base cost)
 * - 30,000 KRW threshold provides safety buffer
 */
export const MIN_DEPOSIT_AMOUNT = 30000;
