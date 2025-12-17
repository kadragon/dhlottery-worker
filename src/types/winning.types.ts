/**
 * Winning Result Types
 *
 * Trace:
 *   spec_id: SPEC-WINNING-001
 *   task_id: TASK-006
 */

export interface WinningResult {
  roundNumber: number;
  rank: number;
  prizeAmount: number;
  matchCount?: number;
}
