/**
 * Main Orchestration Module
 *
 * Trace:
 *   spec_id: SPEC-ORCH-001, SPEC-GHACTION-001
 *   task_id: TASK-008, TASK-011, TASK-GHACTION-001
 */

import { TOTAL_PURCHASE_COST } from './constants';
import { DHLotteryClient } from './dhlottery/client';
import { sendNotification } from './notify/telegram';

async function notifyOrchestrationError(error: unknown): Promise<void> {
  const message = error instanceof Error ? error.message : String(error);

  await sendNotification({
    type: 'error',
    title: 'Orchestration Error',
    message: `워크플로우 실행 중 오류가 발생했습니다: ${message}`,
  });
}

/**
 * Runs the end-to-end workflow once.
 *
 * Non-throwing by design to avoid unintended retries that could repurchase.
 */
export async function runWorkflow(now: Date = new Date()): Promise<void> {
  const client = new DHLotteryClient();

  try {
    await client.login();

    let canPurchase: boolean;
    try {
      canPurchase = await client.checkDeposit(TOTAL_PURCHASE_COST);
    } catch (error) {
      await notifyOrchestrationError(error);
      return;
    }

    if (canPurchase) {
      await client.reservePensionNextWeek();
      await client.buy();
    }

    await client.checkWinning(now);
  } catch (error) {
    await notifyOrchestrationError(error);
  }
}
