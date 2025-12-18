/**
 * Account Information Retrieval Module
 *
 * Trace:
 *   spec_id: SPEC-ACCOUNT-001
 *   task_id: TASK-003, TASK-011
 */

import { USER_AGENT } from '../constants';
import type { AccountInfo, HttpClient } from '../types';
import { DHLotteryError } from '../utils/errors';

/**
 * DHLottery account page URL
 * Using main page (common.do) instead of myPage.do to avoid 302 redirects
 * and match the behavior of the verified n8n workflow.
 */
const MAIN_PAGE_URL = 'https://www.dhlottery.co.kr/common.do?method=main';
const MY_PAGE_URL = 'https://www.dhlottery.co.kr/myPage.do?method=myPage';

/**
 * Fetch and parse account information
 *
 * Strategy:
 * 1. Fetch Main Page (common.do) first.
 *    - Gets the current lottery round (essential).
 *    - Serves as the "returnUrl" visit after login to stabilize session.
 * 2. Fetch My Page (myPage.do) for balance.
 *    - Primary source for deposit balance.
 *    - If this fails (e.g. 302 redirect), fallback to parsing balance from Main Page header.
 *
 * @param client - HTTP client with authenticated session
 * @returns Account information with balance and current round
 * @throws {DHLotteryError} If fetch fails or parsing fails
 */
export async function getAccountInfo(client: HttpClient): Promise<AccountInfo> {
  // Step 1: Fetch Main Page
  // This is required to get the current round and "finalize" the login session
  console.log(
    JSON.stringify({
      level: 'debug',
      module: 'account',
      message: 'Fetching main page',
      url: MAIN_PAGE_URL,
      cookies: client.cookies, // Log cookies to debug session state
    })
  );

  const mainResponse = await client.fetch(MAIN_PAGE_URL, {
    headers: {
      'User-Agent': USER_AGENT,
    },
  });

  if (mainResponse.status !== 200) {
    throw new DHLotteryError(
      `Failed to fetch main page: HTTP ${mainResponse.status}`,
      'ACCOUNT_FETCH_FAILED'
    );
  }

  const mainHtml = await mainResponse.text('euc-kr');

  // Parse round from Main Page (reliable)
  const currentRound = parseRound(mainHtml);

  // Step 2: Fetch My Page for Balance
  // User explicitly requested to use myPage for balance
  console.log(
    JSON.stringify({
      level: 'debug',
      module: 'account',
      message: 'Fetching my page',
      url: MY_PAGE_URL,
    })
  );

  let balance: number;

  try {
    const myPageResponse = await client.fetch(MY_PAGE_URL, {
      headers: {
        'User-Agent': USER_AGENT,
        Referer: MAIN_PAGE_URL, // Add Referer to mimic browser navigation
      },
    });

    if (myPageResponse.status === 200) {
      const myPageHtml = await myPageResponse.text('euc-kr');
      balance = parseBalance(myPageHtml);
    } else {
      // If 302 or error, throw to trigger fallback
      throw new Error(`HTTP ${myPageResponse.status}`);
    }
  } catch (error) {
    console.warn(
      JSON.stringify({
        level: 'warning',
        module: 'account',
        message: 'Failed to fetch My Page, falling back to Main Page balance',
        error: error instanceof Error ? error.message : String(error),
      })
    );

    // Fallback: Try to parse balance from Main Page header
    balance = parseBalance(mainHtml);
  }

  // Validate parsed data
  validateAccountInfo(balance, currentRound);

  return {
    balance,
    currentRound,
  };
}

/**
 * Parse deposit balance from HTML
 * Actual format: <td class="ta_right" colspan="3">N,NNN 원</td>
 * Note: HTML is EUC-KR encoded, so we match the pattern without Korean characters
 */
function parseBalance(html: string): number {
  // Try multiple regex patterns to match different HTML structures
  const patterns = [
    // Pattern 1: Main Page Header (<li class="money"><a href="..."><strong>N,NNN</strong>원</a></li>)
    /<li[^>]*class="money"[^>]*>[\s\S]*?<strong>([\d,]+)<\/strong>/i,
    // Pattern 2: <td class="ta_right" colspan="3">N,NNN (space before closing)
    /<td[^>]*class="ta_right"[^>]*>\s*([\d,]+)\s+/i,
    // Pattern 3: <strong>N,NNN</strong> (deposit amount in other locations)
    // Made stricter: requires '원' suffix or context to avoid matching random numbers like round
    /<strong>([\d,]+)<\/strong>\s*원/,
  ];

  let match: RegExpMatchArray | null = null;
  for (const pattern of patterns) {
    match = html.match(pattern);
    if (match) {
      break;
    }
  }

  if (!match) {
    // Debug: Log context for troubleshooting
    console.log(
      JSON.stringify({
        level: 'error',
        module: 'account',
        message: 'Balance regex did not match any pattern',
        htmlSample: html.substring(1400, 1500),
      })
    );

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
 * Extracts from #lottoDrwNo element (like n8n does)
 */
function parseRound(html: string): number {
  // Try multiple patterns
  const patterns = [
    // Pattern 1: id="lottoDrwNo" element (n8n style)
    /id="lottoDrwNo"[^>]*>(\d+)/i,
    // Pattern 2: Korean text format
    /제<strong>(\d+)<\/strong>회/,
    /제\s*(\d+)\s*회/,
    // Pattern 3: Table format
    /<td>(\d{4})<\/td>/,
  ];

  for (const pattern of patterns) {
    const match = html.match(pattern);
    if (match) {
      const round = Number.parseInt(match[1], 10);
      if (!Number.isNaN(round) && round > 0) {
        console.log(
          JSON.stringify({
            level: 'info',
            module: 'account',
            message: 'Found current round from HTML',
            round,
            pattern: pattern.source,
          })
        );

        // Return next round (current + 1 for upcoming purchase)
        return round + 1;
      }
    }
  }

  throw new DHLotteryError(
    'Failed to parse lottery round from account page',
    'ACCOUNT_PARSE_ROUND_FAILED'
  );
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
