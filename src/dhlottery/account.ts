/**
 * Account Information Retrieval Module
 *
 * Trace:
 *   spec_id: SPEC-ACCOUNT-001
 *   task_id: TASK-003, TASK-011
 */

import type { AccountInfo, HttpClient } from '../types';
import { DHLotteryError } from '../utils/errors';

/**
 * DHLottery account page URL
 */
const ACCOUNT_URL = 'https://www.dhlottery.co.kr/myPage.do?method=myPage';

/**
 * Fetch and parse account information
 *
 * @param client - HTTP client with authenticated session
 * @returns Account information with balance and current round
 * @throws {DHLotteryError} If fetch fails or parsing fails
 */
export async function getAccountInfo(client: HttpClient): Promise<AccountInfo> {
  // Fetch account page
  const response = await client.fetch(ACCOUNT_URL);

  // Check HTTP status
  if (response.status !== 200) {
    throw new DHLotteryError(
      `Failed to fetch account page: HTTP ${response.status}`,
      'ACCOUNT_FETCH_FAILED'
    );
  }

  // Get HTML content
  const html = await response.text();

  // Parse deposit balance
  const balance = parseBalance(html);

  // Parse current lottery round
  const currentRound = parseRound(html);

  // Validate parsed data
  validateAccountInfo(balance, currentRound);

  return {
    balance,
    currentRound,
  };
}

/**
 * Parse deposit balance from HTML
 * Format: <dd><strong>N,NNN</strong>원</dd>
 */
function parseBalance(html: string): number {
  // Regex to match balance pattern: number with optional commas followed by 원
  const balanceRegex = /<dd><strong>([\d,]+)<\/strong>원<\/dd>/;
  const match = html.match(balanceRegex);

  if (!match) {
    throw new DHLotteryError(
      'Failed to parse balance from account page',
      'ACCOUNT_PARSE_BALANCE_FAILED'
    );
  }

  // Remove commas and parse to number
  const balanceStr = match[1].replace(/,/g, '');
  const balance = Number.parseInt(balanceStr, 10);

  if (Number.isNaN(balance)) {
    throw new DHLotteryError(`Invalid balance value: ${match[1]}`, 'ACCOUNT_PARSE_BALANCE_FAILED');
  }

  return balance;
}

/**
 * Parse current lottery round from HTML
 * Format: 제<strong>NNNN</strong>회
 */
function parseRound(html: string): number {
  // Regex to match round pattern
  const roundRegex = /제<strong>(\d+)<\/strong>회/;
  const match = html.match(roundRegex);

  if (!match) {
    throw new DHLotteryError(
      'Failed to parse lottery round from account page',
      'ACCOUNT_PARSE_ROUND_FAILED'
    );
  }

  const round = Number.parseInt(match[1], 10);

  if (Number.isNaN(round)) {
    throw new DHLotteryError(`Invalid round value: ${match[1]}`, 'ACCOUNT_PARSE_ROUND_FAILED');
  }

  return round;
}

/**
 * Validate account information
 */
function validateAccountInfo(balance: number, currentRound: number): void {
  // Balance must be non-negative
  if (balance < 0) {
    throw new DHLotteryError(`Invalid balance: ${balance} (must be >= 0)`, 'ACCOUNT_INVALID_DATA');
  }

  // Round must be positive
  if (currentRound <= 0) {
    throw new DHLotteryError(
      `Invalid round: ${currentRound} (must be > 0)`,
      'ACCOUNT_INVALID_DATA'
    );
  }
}
