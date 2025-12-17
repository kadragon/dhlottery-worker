/**
 * Lottery Purchase Module
 *
 * Trace:
 *   spec_id: SPEC-PURCHASE-001
 *   task_id: TASK-005, TASK-011
 *
 * Implements automatic lottery purchase with 5 games
 */

import { sendNotification } from '../notify/telegram';
import type {
  GameSelection,
  HttpClient,
  PurchaseEnv,
  PurchaseOutcome,
  PurchaseReadyResponse,
  PurchaseResult,
} from '../types';
import { PURCHASE_CONSTANTS } from '../types/purchase.types';
import { getAccountInfo } from './account';

const BASE_URL = 'https://ol.dhlottery.co.kr/olotto/game';

/**
 * Prepares purchase session by calling ready endpoint
 */
async function preparePurchase(): Promise<PurchaseReadyResponse> {
  const response = await fetch(`${BASE_URL}/egovUserReadySocket.json`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json; charset=UTF-8',
    },
  });

  if (!response.ok) {
    throw new Error(`Purchase ready failed: ${response.status}`);
  }

  return await response.json();
}

/**
 * Executes lottery purchase with auto-generated numbers
 */
async function executePurchase(
  readyResponse: PurchaseReadyResponse,
  roundNumber: number
): Promise<PurchaseResult> {
  // Generate 5 game selections with auto mode
  const games: GameSelection[] = ['A', 'B', 'C', 'D', 'E'].map((alpabet) => ({
    genType: '0', // Auto-generated numbers
    arrGameChoiceNum: null,
    alpabet,
  }));

  const params = new URLSearchParams({
    round: roundNumber.toString(),
    direct: readyResponse.ready_ip,
    nBuyAmount: PURCHASE_CONSTANTS.TOTAL_COST.toString(),
    param: JSON.stringify(games),
    ROUND_DRAW_DATE: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toLocaleDateString('en-CA'),
    WAMT_PAY_TLMT_END_DT: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toLocaleDateString(
      'en-CA'
    ),
    gameCnt: PURCHASE_CONSTANTS.GAME_COUNT.toString(),
  });

  const response = await fetch(`${BASE_URL}/execBuy.do`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
    },
    body: params.toString(),
  });

  if (!response.ok) {
    throw new Error(`Purchase execution failed: ${response.status}`);
  }

  return await response.json();
}

/**
 * Formats Korean number with thousands separator
 */
function formatKoreanNumber(amount: number): string {
  return amount.toLocaleString('ko-KR');
}

/**
 * Purchases lottery tickets with automatic number generation
 *
 * @param client - HTTP client with active session
 * @param env - Environment variables including credentials and Telegram config
 * @returns Purchase outcome with success/failure details
 */
export async function purchaseLottery(
  client: HttpClient,
  env: PurchaseEnv
): Promise<PurchaseOutcome> {
  try {
    // Get current lottery round number
    const accountInfo = await getAccountInfo(client);
    const roundNumber = accountInfo.currentRound;

    // Step 1: Prepare purchase session
    const readyResponse = await preparePurchase();

    // Step 2: Execute purchase
    const purchaseResult = await executePurchase(readyResponse, roundNumber);

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
      await sendNotification(
        {
          type: 'success',
          title: '로또 구매 완료',
          message: `${roundNumber}회 로또 ${PURCHASE_CONSTANTS.GAME_COUNT}게임을 ${formatKoreanNumber(PURCHASE_CONSTANTS.TOTAL_COST)}원에 구매했습니다.`,
          details: {
            회차: `${roundNumber}회`,
            게임수: `${PURCHASE_CONSTANTS.GAME_COUNT}게임`,
            결제금액: `${formatKoreanNumber(PURCHASE_CONSTANTS.TOTAL_COST)}원`,
          },
        },
        env
      );

      return successResult;
    }

    // Purchase failed with error code
    const failureResult: PurchaseOutcome = {
      success: false,
      error: purchaseResult.result.resultMsg,
      code: purchaseResult.result.resultCode,
    };

    // Send error notification
    await sendNotification(
      {
        type: 'error',
        title: '로또 구매 실패',
        message: purchaseResult.result.resultMsg,
        details: {
          오류코드: purchaseResult.result.resultCode,
        },
      },
      env
    );

    return failureResult;
  } catch (error) {
    // Network or unexpected error
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';

    const failureResult: PurchaseOutcome = {
      success: false,
      error: errorMessage,
    };

    // Send error notification
    await sendNotification(
      {
        type: 'error',
        title: '로또 구매 실패',
        message: `구매 중 오류가 발생했습니다: ${errorMessage}`,
      },
      env
    );

    return failureResult;
  }
}
