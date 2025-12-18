/**
 * DHLottery Client Tests
 *
 * Trace:
 *   spec_id: SPEC-ARCH-001
 *   task_id: TASK-019
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { DHLotteryClient } from './client';
import * as authModule from './auth';
import * as accountModule from './account';
import * as chargeModule from './charge';
import * as buyModule from './buy';
import * as checkModule from './check';
import type { WorkerEnv } from '../types';

// Mock dependencies
vi.mock('./auth');
vi.mock('./account');
vi.mock('./charge');
vi.mock('./buy');
vi.mock('./check');
vi.mock('../client/http', () => ({
  createHttpClient: vi.fn(() => ({
    fetch: vi.fn(),
    cookies: {},
    getCookieHeader: vi.fn(),
  })),
}));

describe('DHLotteryClient', () => {
  let client: DHLotteryClient;
  let mockEnv: WorkerEnv;

  beforeEach(() => {
    mockEnv = {
      USER_ID: 'test_user',
      PASSWORD: 'test_password',
      TELEGRAM_BOT_TOKEN: 'test_token',
      TELEGRAM_CHAT_ID: 'test_chat_id',
    };
    client = new DHLotteryClient(mockEnv);
  });

  it('TEST-ARCH-001: should instantiate correctly', () => {
    expect(client).toBeInstanceOf(DHLotteryClient);
  });

  it('TEST-ARCH-002: should delegate login to auth module', async () => {
    await client.login();
    expect(authModule.login).toHaveBeenCalledWith(expect.anything(), mockEnv);
  });

  it('TEST-ARCH-002: should delegate getAccountInfo to account module', async () => {
    await client.getAccountInfo();
    expect(accountModule.getAccountInfo).toHaveBeenCalledWith(expect.anything());
  });

  it('TEST-ARCH-002: should delegate checkDeposit to charge module', async () => {
    await client.checkDeposit();
    expect(chargeModule.checkDeposit).toHaveBeenCalledWith(expect.anything(), mockEnv);
  });

  it('TEST-ARCH-002: should delegate buy to buy module', async () => {
    await client.buy();
    expect(buyModule.purchaseLottery).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        DHLOTTERY_USER_ID: mockEnv.USER_ID,
      })
    );
  });

  it('TEST-ARCH-002: should delegate checkWinning to check module', async () => {
    const now = new Date();
    await client.checkWinning(now);
    expect(checkModule.checkWinning).toHaveBeenCalledWith(expect.anything(), mockEnv, now);
  });
});
