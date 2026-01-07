/**
 * Account Information Retrieval Module
 *
 * Trace:
 *   spec_id: SPEC-ACCOUNT-001, SPEC-REFACTOR-P2-LOG-001
 *   task_id: TASK-003, TASK-011, TASK-REFACTOR-P2-002
 */

import { USER_AGENT } from '../constants';
import type { AccountInfo, HttpClient } from '../types';
import { DHLotteryError } from '../utils/errors';
import { logger } from '../utils/logger';

/**
 * DHLottery API URLs
 *
 * 2026-01 Update:
 * - Main page (common.do) now redirects to new homepage
 * - Use /lt645/selectThsLt645Info.do API for current lottery round (JSON)
 * - Use /mypage/selectUserMndp.do API for balance (JSON)
 *   - mypage/home is JS-rendered, HTML parsing no longer works
 */
const LOTTO_ROUND_API_URL = 'https://www.dhlottery.co.kr/lt645/selectThsLt645Info.do';
const BALANCE_API_URL = 'https://dhlottery.co.kr/mypage/selectUserMndp.do';

/**
 * Fetch and parse account information
 *
 * Strategy (2026-01 Update):
 * 1. Fetch Lotto Round API (/lt645/selectThsLt645Info.do) first.
 *    - Gets the current lottery round via JSON API.
 *    - Public API, no authentication needed.
 * 2. Fetch Balance API (/mypage/selectUserMndp.do) for balance.
 *    - Gets available balance (crntEntrsAmt) via JSON API.
 *    - Requires authenticated session.
 *
 * @param client - HTTP client with authenticated session
 * @returns Account information with balance and current round
 * @throws {DHLotteryError} If fetch fails or parsing fails
 */
export async function getAccountInfo(client: HttpClient): Promise<AccountInfo> {
  // Step 1: Fetch Lotto Round API for current round
  logger.debug('Fetching lotto round API', {
    module: 'account',
    url: LOTTO_ROUND_API_URL,
  });

  const roundResponse = await client.fetch(LOTTO_ROUND_API_URL, {
    headers: {
      'User-Agent': USER_AGENT,
    },
  });

  if (roundResponse.status !== 200) {
    const location = roundResponse.headers.get('Location');
    const locationInfo = location ? ` (Location: ${location})` : '';
    throw new DHLotteryError(
      `Account fetch failed at round API (url: ${LOTTO_ROUND_API_URL}): HTTP ${roundResponse.status}${locationInfo}`,
      'ACCOUNT_FETCH_FAILED'
    );
  }

  // Parse round from JSON API response
  const currentRound = await parseRoundFromApi(roundResponse);

  // Step 2: Fetch Balance API
  // 2026-01 Update: mypage/home is JS-rendered, use JSON API instead
  logger.debug('Fetching balance API', {
    module: 'account',
    url: BALANCE_API_URL,
  });

  const balanceResponse = await client.fetch(BALANCE_API_URL, {
    headers: {
      'User-Agent': USER_AGENT,
      Referer: 'https://www.dhlottery.co.kr/mypage/home',
    },
  });

  if (balanceResponse.status !== 200) {
    const location = balanceResponse.headers.get('Location');
    const locationInfo = location ? ` (Location: ${location})` : '';
    throw new DHLotteryError(
      `Account fetch failed at balance API (url: ${BALANCE_API_URL}): HTTP ${balanceResponse.status}${locationInfo}`,
      'ACCOUNT_FETCH_FAILED'
    );
  }

  const balance = await parseBalanceFromApi(balanceResponse);

  // Validate parsed data
  validateAccountInfo(balance, currentRound);

  return {
    balance,
    currentRound,
  };
}

/**
 * Balance API Response Type
 *
 * 2026-01 Update: /mypage/selectUserMndp.do returns JSON:
 * {
 *   "resultCode": null,
 *   "resultMessage": null,
 *   "data": {
 *     "userMndp": {
 *       "crntEntrsAmt": 20000,  // Available balance for purchase
 *       "csblDpstAmt": 4237000,
 *       ...
 *     }
 *   }
 * }
 */
interface BalanceApiResponse {
  resultCode: string | null;
  resultMessage: string | null;
  data: {
    userMndp: {
      crntEntrsAmt: number;
      csblDpstAmt?: number;
      csblTkmnyAmt?: number;
      ncsblDpstAmt?: number;
      ncsblTkmnyAmt?: number;
      useDsalAmt?: number;
    };
  };
}

/**
 * Parse balance from JSON API response
 * Uses /mypage/selectUserMndp.do API endpoint
 *
 * 2026-01 Update: mypage/home is JS-rendered, use JSON API instead
 */
async function parseBalanceFromApi(response: { json: () => Promise<unknown> }): Promise<number> {
  try {
    const data = (await response.json()) as BalanceApiResponse;

    if (data?.data?.userMndp?.crntEntrsAmt === undefined) {
      throw new Error('Missing crntEntrsAmt in API response');
    }

    const balance = data.data.userMndp.crntEntrsAmt;

    if (typeof balance !== 'number') {
      throw new Error(`Invalid balance type: ${typeof balance}`);
    }

    logger.debug('Found balance from API', {
      module: 'account',
      balance,
    });

    return balance;
  } catch (error) {
    throw new DHLotteryError(
      `Failed to parse balance from API: ${error instanceof Error ? error.message : String(error)}`,
      'ACCOUNT_PARSE_BALANCE_FAILED'
    );
  }
}

/**
 * Lotto Round API Response Type
 *
 * 2026-01 Update: /lt645/selectThsLt645Info.do returns JSON:
 * {
 *   "resultCode": null,
 *   "resultMessage": null,
 *   "data": {
 *     "result": {
 *       "ltEpsd": 1206,      // Current round number
 *       "ltRflYmd": "20260110",
 *       "ltRflHh": "20",
 *       "ltRflMm": "00"
 *     },
 *     "gameMng": null
 *   }
 * }
 *
 * Trace: spec_id: SPEC-ACCOUNT-003, task_id: TASK-ROUND-API-UPDATE
 */
interface LottoRoundApiResponse {
  resultCode: string | null;
  resultMessage: string | null;
  data: {
    result: {
      ltEpsd: number;
      ltRflYmd: string;
      ltRflHh: string;
      ltRflMm: string;
    };
    gameMng: unknown;
  };
}

/**
 * Parse current lottery round from API JSON response
 * Uses /lt645/selectThsLt645Info.do API endpoint
 */
async function parseRoundFromApi(response: { json: () => Promise<unknown> }): Promise<number> {
  try {
    const data = (await response.json()) as LottoRoundApiResponse;

    if (!data?.data?.result?.ltEpsd) {
      throw new Error('Missing ltEpsd in API response');
    }

    const round = data.data.result.ltEpsd;

    if (typeof round !== 'number' || round <= 0) {
      throw new Error(`Invalid round value: ${round}`);
    }

    logger.debug('Found current round from API', {
      module: 'account',
      round,
      drawDate: data.data.result.ltRflYmd,
    });

    return round;
  } catch (error) {
    throw new DHLotteryError(
      `Failed to parse lottery round from API: ${error instanceof Error ? error.message : String(error)}`,
      'ACCOUNT_PARSE_ROUND_FAILED'
    );
  }
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
