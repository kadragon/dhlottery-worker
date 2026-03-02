/**
 * DHLottery Client Facade
 *
 * Trace:
 *   spec_id: SPEC-ARCH-001, SPEC-GHACTION-001
 *   task_id: TASK-019, TASK-GHACTION-001
 */

import { createHttpClient } from '../client/http';
import { NotificationCollector } from '../notify/notification-collector';
import type {
  AccountInfo,
  HttpClient,
  PensionReserveOutcome,
  PurchaseOutcome,
  WinningResult,
} from '../types';
import { getAccountInfo } from './account';
import { login } from './auth';
import { purchaseLottery } from './buy';
import { checkDeposit } from './charge';
import { checkWinning } from './check';
import { reservePensionNextWeek } from './pension-reserve';

export class DHLotteryClient {
  private client: HttpClient;
  readonly collector: NotificationCollector;

  constructor() {
    this.client = createHttpClient();
    this.collector = new NotificationCollector();
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
  async checkDeposit(requiredAmount?: number): Promise<boolean> {
    return await checkDeposit(this.client, requiredAmount, this.collector);
  }

  /**
   * Reserve next week's pension 720+ ticket (1 round, all groups, 1 ticket each)
   */
  async reservePensionNextWeek(): Promise<PensionReserveOutcome> {
    return await reservePensionNextWeek(this.client, this.collector);
  }

  /**
   * Purchase lottery tickets (5 games, auto)
   */
  async buy(): Promise<PurchaseOutcome> {
    return await purchaseLottery(this.client, this.collector);
  }

  /**
   * Check winning results for the previous week
   */
  async checkWinning(now: Date = new Date()): Promise<WinningResult[]> {
    return await checkWinning(this.client, now, this.collector);
  }
}
