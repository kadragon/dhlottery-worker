/**
 * Lottery Purchase Module
 *
 * Trace:
 *   spec_id: SPEC-PURCHASE-001, SPEC-REFACTOR-P2-ERROR-001, SPEC-GHACTION-001
 *   task_id: TASK-005, TASK-011, TASK-REFACTOR-P2-003, TASK-GHACTION-001
 *
 * Implements automatic lottery purchase with 5 games
 */

import { USER_AGENT } from '../constants';
import { sendNotification } from '../notify/telegram';
import type {
  GameSelection,
  HttpClient,
  PurchaseOutcome,
  PurchaseReadyResponse,
  PurchaseResult,
} from '../types';
import { PURCHASE_CONSTANTS } from '../types/purchase.types';
import { getNextSaturdayKst } from '../utils/date';
import { PurchaseError } from '../utils/errors';
import { formatKoreanNumber } from '../utils/format';
import { logger } from '../utils/logger';
import { getAccountInfo } from './account';

const BASE_URL = 'https://ol.dhlottery.co.kr/olotto/game';
const GAME_PAGE_URL = `${BASE_URL}/game645.do`;
const AJAX_HEADERS = {
  Origin: 'https://ol.dhlottery.co.kr',
  Referer: GAME_PAGE_URL,
  'X-Requested-With': 'XMLHttpRequest',
} as const;

/**
 * Prepares purchase session by calling ready endpoint
 */
async function preparePurchase(client: HttpClient): Promise<PurchaseReadyResponse> {
  const response = await client.fetch(`${BASE_URL}/egovUserReadySocket.json`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=UTF-8',
      'User-Agent': USER_AGENT,
      ...AJAX_HEADERS,
    },
  });

  if (response.status !== 200) {
    throw new PurchaseError(`Purchase ready failed: ${response.status}`, 'PURCHASE_READY_FAILED');
  }

  return await response.json();
}

/**
 * Executes lottery purchase with auto-generated numbers
 */
async function executePurchase(
  client: HttpClient,
  readyResponse: PurchaseReadyResponse,
  roundNumber: number
): Promise<PurchaseResult> {
  const drawDate = getNextSaturdayKst();
  const payLimitDate = addYearsAndDays(drawDate, 1, 1);

  // Generate 5 game selections with auto mode
  const games: GameSelection[] = ['A', 'B', 'C', 'D', 'E'].map((alpabet) => ({
    genType: '0', // Auto-generated numbers
    arrGameChoiceNum: null,
    alpabet,
  }));

  // n8n style: only send required parameters
  const params = new URLSearchParams({
    round: roundNumber.toString(),
    direct: readyResponse.ready_ip,
    nBuyAmount: PURCHASE_CONSTANTS.TOTAL_COST.toString(),
    param: JSON.stringify(games),
    gameCnt: PURCHASE_CONSTANTS.GAME_COUNT.toString(),
    saleMdaDcd: '10',
    ROUND_DRAW_DATE: formatDateWithSlashes(drawDate),
    WAMT_PAY_TLMT_END_DT: formatDateWithSlashes(payLimitDate),
  });

  logger.debug('Purchase parameters', {
    module: 'buy',
    round: roundNumber,
    direct: readyResponse.ready_ip,
    nBuyAmount: PURCHASE_CONSTANTS.TOTAL_COST,
    gameCnt: PURCHASE_CONSTANTS.GAME_COUNT,
  });

  const response = await client.fetch(`${BASE_URL}/execBuy.do`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
      'User-Agent': USER_AGENT,
      ...AJAX_HEADERS,
    },
    body: params.toString(),
  });

  if (response.status !== 200) {
    throw new PurchaseError(
      `Purchase execution failed: ${response.status}`,
      'PURCHASE_EXECUTION_FAILED'
    );
  }

  return await response.json();
}

function formatDateWithSlashes(date: string): string {
  return date.replace(/-/g, '/');
}

function addYearsAndDays(date: string, years: number, days: number): string {
  const [year, month, day] = date.split('-').map((part) => Number(part));
  const base = new Date(Date.UTC(year, month - 1, day));
  base.setUTCFullYear(base.getUTCFullYear() + years);
  base.setUTCDate(base.getUTCDate() + days);
  const resultYear = base.getUTCFullYear();
  const resultMonth = String(base.getUTCMonth() + 1).padStart(2, '0');
  const resultDay = String(base.getUTCDate()).padStart(2, '0');
  return `${resultYear}-${resultMonth}-${resultDay}`;
}

/**
 * Purchases lottery tickets with automatic number generation
 *
 * @param client - HTTP client with active session
 * @returns Purchase outcome with success/failure details
 */
export async function purchaseLottery(client: HttpClient): Promise<PurchaseOutcome> {
  try {
    // Get current lottery round number
    const accountInfo = await getAccountInfo(client);
    const roundNumber = accountInfo.currentRound;

    // Step 1: Prepare purchase session
    const readyResponse = await preparePurchase(client);

    // Step 2: Execute purchase
    const purchaseResult = await executePurchase(client, readyResponse, roundNumber);

    // Step 3: Check result
    if (purchaseResult.result.resultCode === PURCHASE_CONSTANTS.SUCCESS_CODE) {
      // Success
      const successResult: PurchaseOutcome = {
        success: true,
        roundNumber,
        gameCount: PURCHASE_CONSTANTS.GAME_COUNT,
        totalAmount: PURCHASE_CONSTANTS.TOTAL_COST,
        purchaseDate: new Date().toISOString(),
        message: purchaseResult.result.resultMsg,
      };

      // Send success notification
      await sendNotification({
        type: 'success',
        title: 'Lottery Purchase Completed',
        message: `${roundNumber}회 로또 ${PURCHASE_CONSTANTS.GAME_COUNT}게임을 ${formatKoreanNumber(PURCHASE_CONSTANTS.TOTAL_COST)}원에 구매했습니다.`,
        details: {
          회차: `${roundNumber}회`,
          게임수: `${PURCHASE_CONSTANTS.GAME_COUNT}게임`,
          결제금액: `${formatKoreanNumber(PURCHASE_CONSTANTS.TOTAL_COST)}원`,
        },
      });

      return successResult;
    }

    // Purchase failed with error code
    const failureResult: PurchaseOutcome = {
      success: false,
      error: purchaseResult.result.resultMsg,
      code: purchaseResult.result.resultCode,
    };

    // Send error notification
    await sendNotification({
      type: 'error',
      title: 'Lottery Purchase Failed',
      message: purchaseResult.result.resultMsg,
      details: {
        오류코드: purchaseResult.result.resultCode,
      },
    });

    return failureResult;
  } catch (error) {
    // Network or unexpected error
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';

    const failureResult: PurchaseOutcome = {
      success: false,
      error: errorMessage,
    };

    // Send error notification
    await sendNotification({
      type: 'error',
      title: 'Lottery Purchase Failed',
      message: `구매 중 오류가 발생했습니다: ${errorMessage}`,
    });

    return failureResult;
  }
}
