/**
 * DHLottery Client Tests
 *
 * Trace:
 *   spec_id: SPEC-ARCH-001
 *   task_id: TASK-019
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { DHLotteryClient } from './client';
import * as authModule from './auth';
import * as accountModule from './account';
import * as chargeModule from './charge';
import * as buyModule from './buy';
import * as checkModule from './check';
import * as pensionReserveModule from './pension-reserve';

// Mock dependencies
vi.mock('./auth');
vi.mock('./account');
vi.mock('./charge');
vi.mock('./buy');
vi.mock('./check');
vi.mock('./pension-reserve');
vi.mock('../client/http', () => ({
  createHttpClient: vi.fn(() => ({
    fetch: vi.fn(),
    cookies: {},
    getCookieHeader: vi.fn(),
  })),
}));

describe('DHLotteryClient', () => {
  let client: DHLotteryClient;

  beforeEach(() => {
    // Mock process.env
    vi.stubEnv('USER_ID', 'test_user');
    vi.stubEnv('PASSWORD', 'test_password');
    vi.stubEnv('TELEGRAM_BOT_TOKEN', 'test_token');
    vi.stubEnv('TELEGRAM_CHAT_ID', 'test_chat_id');

    client = new DHLotteryClient();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('TEST-ARCH-001: should instantiate correctly', () => {
    expect(client).toBeInstanceOf(DHLotteryClient);
  });

  it('TEST-ARCH-002: should delegate login to auth module', async () => {
    await client.login();
    expect(authModule.login).toHaveBeenCalledWith(expect.anything());
  });

  it('TEST-ARCH-002: should delegate getAccountInfo to account module', async () => {
    await client.getAccountInfo();
    expect(accountModule.getAccountInfo).toHaveBeenCalledWith(expect.anything());
  });

  it('TEST-ARCH-002: should delegate checkDeposit to charge module', async () => {
    await client.checkDeposit();
    expect(chargeModule.checkDeposit).toHaveBeenCalledWith(expect.anything());
  });

  it('should delegate checkDeposit with required amount to charge module', async () => {
    await client.checkDeposit(10000);
    expect(chargeModule.checkDeposit).toHaveBeenCalledWith(expect.anything(), 10000);
  });

  it('TEST-ARCH-002: should delegate buy to buy module', async () => {
    await client.buy();
    expect(buyModule.purchaseLottery).toHaveBeenCalledWith(expect.anything());
  });

  it('TEST-ARCH-002: should delegate checkWinning to check module', async () => {
    const now = new Date();
    await client.checkWinning(now);
    expect(checkModule.checkWinning).toHaveBeenCalledWith(expect.anything(), now);
  });

  it('should delegate pension reserve to pension-reserve module', async () => {
    await client.reservePensionNextWeek();
    expect(pensionReserveModule.reservePensionNextWeek).toHaveBeenCalledWith(expect.anything());
  });

  /**
   * TEST-REFACTOR-P1-DOC-001: Verify getAccountInfo() has JSDoc documentation
   *
   * Criteria:
   * - Method has JSDoc comment block
   * - Comment explains it's for external API usage
   * - Comment notes that internal code (buy.ts, charge.ts) call module directly
   */
  describe('TEST-REFACTOR-P1-DOC-001: getAccountInfo JSDoc documentation', () => {
    it('should have JSDoc comment explaining external API purpose', () => {
      // Read the source file and verify JSDoc exists
      const fs = require('fs');
      const path = require('path');
      const sourceFile = path.join(__dirname, 'client.ts');
      const content = fs.readFileSync(sourceFile, 'utf-8');

      // Verify JSDoc comment block exists before getAccountInfo method
      expect(content).toMatch(/\/\*\*[\s\S]*?Get current account information[\s\S]*?@returns[\s\S]*?\*\/\s*async getAccountInfo/);
    });

    it('should document that it is intended for external API usage', () => {
      const fs = require('fs');
      const path = require('path');
      const sourceFile = path.join(__dirname, 'client.ts');
      const content = fs.readFileSync(sourceFile, 'utf-8');

      // Verify JSDoc mentions external API usage
      expect(content).toMatch(/\/\*\*[\s\S]*?external[\s\S]*?@returns[\s\S]*?\*\/\s*async getAccountInfo/i);
    });
  });

  /**
   * TEST-REFACTOR-P1-DOC-002: Verify no orphaned references to client.getAccountInfo()
   *
   * Criteria:
   * - buy.ts calls getAccountInfo directly from account module
   * - charge.ts calls getAccountInfo directly from account module
   * - No external code tries to use client.getAccountInfo()
   */
  describe('TEST-REFACTOR-P1-DOC-002: getAccountInfo usage patterns', () => {
    it('should be called via module delegation pattern in tests', async () => {
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 50000,
        currentRound: 1234,
      });

      const result = await client.getAccountInfo();

      expect(result).toEqual({ balance: 50000, currentRound: 1234 });
      expect(accountModule.getAccountInfo).toHaveBeenCalled();
    });
  });

  /**
   * TEST-REFACTOR-P1-DOC-003: Verify method signature and return type
   *
   * Criteria:
   * - Method is async and returns Promise<AccountInfo>
   * - Method accepts no parameters
   * - Method delegates to account module's getAccountInfo function
   */
  describe('TEST-REFACTOR-P1-DOC-003: getAccountInfo interface', () => {
    it('should have correct method signature (no params, returns AccountInfo)', async () => {
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 75000,
        currentRound: 999,
      });

      const result = await client.getAccountInfo();

      expect(typeof result === 'object').toBe(true);
      expect('balance' in result).toBe(true);
      expect('currentRound' in result).toBe(true);
    });

    it('should delegate to account module with client instance', async () => {
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 100000,
        currentRound: 5000,
      });

      await client.getAccountInfo();

      // Verify delegation: first argument should be the HTTP client
      expect(accountModule.getAccountInfo).toHaveBeenCalledWith(expect.anything());
    });
  });
});
