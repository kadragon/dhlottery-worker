/**
 * Cloudflare Workers Entry Point (scheduled-only)
 *
 * Trace:
 *   spec_id: SPEC-ORCH-001
 *   task_id: TASK-008
 */

import type { ExecutionContext, ScheduledController } from '@cloudflare/workers-types';
import { createHttpClient } from './client/http';
import { login } from './dhlottery/auth';
import { purchaseLottery } from './dhlottery/buy';
import { checkDeposit } from './dhlottery/charge';
import { checkWinning } from './dhlottery/check';
import { sendNotification } from './notify/telegram';
import type { AuthEnv } from './types/auth.types';
import type { TelegramEnv } from './types/notification.types';
import type { PurchaseEnv } from './types/purchase.types';

export type WorkerEnv = AuthEnv & TelegramEnv;

function toPurchaseEnv(env: WorkerEnv): PurchaseEnv {
  return {
    DHLOTTERY_USER_ID: env.USER_ID,
    DHLOTTERY_USER_PW: env.PASSWORD,
    TELEGRAM_BOT_TOKEN: env.TELEGRAM_BOT_TOKEN,
    TELEGRAM_CHAT_ID: env.TELEGRAM_CHAT_ID,
  };
}

async function notifyOrchestrationError(env: WorkerEnv, error: unknown): Promise<void> {
  const message = error instanceof Error ? error.message : String(error);

  await sendNotification(
    {
      type: 'error',
      title: 'Orchestration Error',
      message: `워크플로우 실행 중 오류가 발생했습니다: ${message}`,
    },
    env
  );
}

/**
 * Runs the end-to-end workflow once.
 *
 * Non-throwing by design to avoid unintended retries that could repurchase.
 */
export async function runWorkflow(env: WorkerEnv, now: Date = new Date()): Promise<void> {
  const client = createHttpClient();

  try {
    await login(client, env);

    let canPurchase: boolean;
    try {
      canPurchase = await checkDeposit(client, env);
    } catch (error) {
      await notifyOrchestrationError(env, error);
      return;
    }

    if (canPurchase) {
      await purchaseLottery(client, toPurchaseEnv(env));
    }

    await checkWinning(client, env, now);
  } catch (error) {
    await notifyOrchestrationError(env, error);
  }
}

export default {
  async scheduled(
    _event: ScheduledController,
    env: WorkerEnv,
    ctx: ExecutionContext
  ): Promise<void> {
    ctx.waitUntil(runWorkflow(env));
  },
};
