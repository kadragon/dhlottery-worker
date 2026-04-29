/**
 * Main Orchestration Module
 *
 * Trace:
 *   spec_id: SPEC-ORCH-001, SPEC-GHACTION-001
 *   task_id: TASK-008, TASK-011, TASK-GHACTION-001
 */

import { WEEKLY_COMBINED_REQUIRED_BALANCE } from './constants';
import { DHLotteryClient } from './dhlottery/client';
import { sendCombinedNotification } from './notify/telegram';

/**
 * Runs the end-to-end workflow once.
 *
 * Non-throwing by design to avoid unintended retries that could repurchase.
 * All notifications are collected and sent as a single Telegram message at the end.
 * Returns false if the Telegram send fails after all retries; true otherwise.
 */
export async function runWorkflow(now: Date = new Date()): Promise<boolean> {
  const client = new DHLotteryClient();

  try {
    await client.login();

    let canPurchase: boolean;
    try {
      canPurchase = await client.checkDeposit(WEEKLY_COMBINED_REQUIRED_BALANCE);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      client.collector.add({
        type: 'error',
        title: 'Orchestration Error',
        message: `워크플로우 실행 중 오류가 발생했습니다: ${message}`,
      });
      canPurchase = false;
    }

    if (canPurchase) {
      await client.reservePensionNextWeek();
      await client.buy();
    }

    await client.checkWinning(now);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    client.collector.add({
      type: 'error',
      title: 'Orchestration Error',
      message: `워크플로우 실행 중 오류가 발생했습니다: ${message}`,
    });
  }

  // Single send at the end; propagate delivery outcome to caller
  if (!client.collector.isEmpty()) {
    return sendCombinedNotification(client.collector.getPayloads());
  }
  return true;
}
