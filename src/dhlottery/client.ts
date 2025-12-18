/**
 * DHLottery Client Facade
 *
 * Trace:
 *   spec_id: SPEC-ARCH-001
 *   task_id: TASK-019
 */

import { createHttpClient } from '../client/http';
import type { AccountInfo, HttpClient, PurchaseOutcome, WinningResult, WorkerEnv } from '../types';
import { getAccountInfo } from './account';
import { login } from './auth';
import { purchaseLottery } from './buy';
import { checkDeposit } from './charge';
import { checkWinning } from './check';

export class DHLotteryClient {
  private client: HttpClient;
  private env: WorkerEnv;

  constructor(env: WorkerEnv) {
    this.env = env;
    this.client = createHttpClient();
  }

  /**
   * Log in to DHLottery
   */
  async login(): Promise<void> {
    await login(this.client, this.env);
  }

  /**
   * Get current account information (balance, round)
   */
  async getAccountInfo(): Promise<AccountInfo> {
    return await getAccountInfo(this.client);
  }

  /**
   * Check deposit balance and initialize charge if needed
   * @returns true if balance is sufficient, false otherwise
   */
  async checkDeposit(): Promise<boolean> {
    return await checkDeposit(this.client, this.env);
  }

  /**
   * Purchase lottery tickets (5 games, auto)
   */
  async buy(): Promise<PurchaseOutcome> {
    return await purchaseLottery(this.client, {
      DHLOTTERY_USER_ID: this.env.USER_ID,
      DHLOTTERY_USER_PW: this.env.PASSWORD,
      TELEGRAM_BOT_TOKEN: this.env.TELEGRAM_BOT_TOKEN,
      TELEGRAM_CHAT_ID: this.env.TELEGRAM_CHAT_ID,
    });
  }

  /**
   * Check winning results for the previous week
   */
  async checkWinning(now: Date = new Date()): Promise<WinningResult[]> {
    return await checkWinning(this.client, this.env, now);
  }
}
