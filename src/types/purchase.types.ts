/**
 * Purchase Type Definitions
 *
 * Trace:
 *   spec_id: SPEC-PURCHASE-001
 *   task_id: TASK-005
 */

/**
 * Environment variables for lottery purchase
 */
export interface PurchaseEnv {
  DHLOTTERY_USER_ID: string;
  DHLOTTERY_USER_PW: string;
  TELEGRAM_BOT_TOKEN: string;
  TELEGRAM_CHAT_ID: string;
}

/**
 * Purchase ready response from /egovUserReadySocket.json
 */
export interface PurchaseReadyResponse {
  direct_yn: string;
  ready_ip: string;
  ready_time: string;
  ready_cnt: string;
}

/**
 * Individual game selection for purchase
 */
export interface GameSelection {
  genType: string; // "0" for auto-generated numbers
  arrGameChoiceNum: number[] | null; // null for auto-generated
  alpabet: string; // "A", "B", "C", "D", "E"
}

/**
 * Purchase execution result
 */
export interface PurchaseResult {
  loginYn: string; // "Y" or "N"
  result: {
    resultCode: string; // "100" for success, negative for errors
    resultMsg: string;
  };
}

/**
 * Successful purchase details
 */
export interface PurchaseSuccess {
  success: true;
  roundNumber: number;
  gameCount: number;
  totalAmount: number;
  purchaseDate: string;
  message: string;
}

/**
 * Failed purchase details
 */
export interface PurchaseFailure {
  success: false;
  error: string;
  code?: string;
}

/**
 * Purchase operation result (success or failure)
 */
export type PurchaseOutcome = PurchaseSuccess | PurchaseFailure;

/**
 * Business constants for lottery purchase
 */
export const PURCHASE_CONSTANTS = {
  GAME_COUNT: 5,
  COST_PER_GAME: 1000,
  TOTAL_COST: 5000,
  MODE: 'auto',
  SUCCESS_CODE: '100',
} as const;
