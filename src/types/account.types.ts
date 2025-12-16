/**
 * Account Information Types
 *
 * Trace:
 *   spec_id: SPEC-ACCOUNT-001
 *   task_id: TASK-003
 */

/**
 * Account information from DHLottery
 */
export interface AccountInfo {
  /**
   * Current deposit balance in KRW
   * Must be >= 0
   */
  balance: number;

  /**
   * Current lottery round number
   * Must be > 0
   */
  currentRound: number;
}
