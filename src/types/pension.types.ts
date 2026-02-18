/**
 * Pension 720+ Reserve Type Definitions
 */

export interface ElEncryptedResponse {
  q: string;
}

export interface ElResultBase {
  resultCode: string;
  resultMsg: string;
}

export interface ElRoundRemainTimeResponse extends ElResultBase {
  ROUND: string;
  DRAW_DATE: string;
}

export interface ElDepositResponse extends ElResultBase {
  deposit: string;
}

export interface ElDuplicateRound {
  doubleRound: string;
  doubleCnt: string;
}

export interface ElCheckMyReserveResponse extends ElResultBase {
  doubleRound: ElDuplicateRound[];
}

export interface ElAddMyReserveResponse extends ElResultBase {
  reserveOrderNo?: string;
  reserveOrderDate?: string;
}

export interface PensionReserveSuccess {
  status: 'success';
  success: true;
  skipped: false;
  targetRound: number;
  totalAmount: number;
  ticketCount: number;
  message: string;
  reserveOrderNo?: string;
  reserveOrderDate?: string;
}

export interface PensionReserveSkipped {
  status: 'skipped';
  success: true;
  skipped: true;
  targetRound: number;
  totalAmount: number;
  ticketCount: number;
  message: string;
  duplicateRounds: string[];
}

export interface PensionReserveFailure {
  status: 'failure';
  success: false;
  skipped: false;
  targetRound?: number;
  error: string;
  code?: string;
}

export type PensionReserveOutcome =
  | PensionReserveSuccess
  | PensionReserveSkipped
  | PensionReserveFailure;
