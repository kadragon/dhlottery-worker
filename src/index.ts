/**
 * Cloudflare Workers Entry Point (scheduled-only)
 *
 * Trace:
 *   spec_id: SPEC-ORCH-001
 *   task_id: TASK-008, TASK-011
 */

import type { ExecutionContext, ScheduledController } from '@cloudflare/workers-types';
import { DHLotteryClient } from './dhlottery/client';
import { sendNotification } from './notify/telegram';
import type { WorkerEnv } from './types';

export type { WorkerEnv } from './types';

/**
 * Validate required environment variables
 * Throws an error if any required variable is missing
 */
function validateEnv(env: WorkerEnv): void {
  const required = ['USER_ID', 'PASSWORD', 'TELEGRAM_BOT_TOKEN', 'TELEGRAM_CHAT_ID'] as const;
  const missing = required.filter((key) => !env[key]);

  if (missing.length > 0) {
    throw new Error(`Missing required environment variables: ${missing.join(', ')}`);
  }
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
  const client = new DHLotteryClient(env);

  try {
    await client.login();

    let canPurchase: boolean;
    try {
      canPurchase = await client.checkDeposit();
    } catch (error) {
      await notifyOrchestrationError(env, error);
      return;
    }

    if (canPurchase) {
      await client.buy();
    }

    await client.checkWinning(now);
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
    validateEnv(env);
    ctx.waitUntil(runWorkflow(env));
  },
};
