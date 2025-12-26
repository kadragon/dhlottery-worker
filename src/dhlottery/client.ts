/**
 * DHLottery Client Facade
 *
 * Trace:
 *   spec_id: SPEC-ARCH-001, SPEC-GHACTION-001
 *   task_id: TASK-019, TASK-GHACTION-001
 */

import { createHttpClient } from '../client/http';
import type { AccountInfo, HttpClient, PurchaseOutcome, WinningResult } from '../types';
import { getAccountInfo } from './account';
import { login } from './auth';
import { purchaseLottery } from './buy';
import { checkDeposit } from './charge';
import { checkWinning } from './check';

export class DHLotteryClient {
  private client: HttpClient;

  constructor() {
    this.client = createHttpClient();
  }

  /**
   * Log in to DHLottery
   */
  async login(): Promise<void> {
    await login(this.client);
  }

  /**
   * Get current account information (balance, round)
   *
   * Public API for external callers. Internal code (buy.ts, charge.ts) calls
   * the getAccountInfo() function from the account module directly for better
   * encapsulation and to avoid adding a dependency layer for internal flows.
   * This method exists to provide a unified client interface for external API usage.
   *
   * @returns Account information with balance and current lottery round
   */
  async getAccountInfo(): Promise<AccountInfo> {
    return await getAccountInfo(this.client);
  }

  /**
   * Check deposit balance and initialize charge if needed
   * @returns true if balance is sufficient, false otherwise
   */
  async checkDeposit(): Promise<boolean> {
    return await checkDeposit(this.client);
  }

  /**
   * Purchase lottery tickets (5 games, auto)
   */
  async buy(): Promise<PurchaseOutcome> {
    return await purchaseLottery(this.client);
  }

  /**
   * Check winning results for the previous week
   */
  async checkWinning(now: Date = new Date()): Promise<WinningResult[]> {
    return await checkWinning(this.client, now);
  }
}
