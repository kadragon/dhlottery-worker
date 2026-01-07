/**
 * Deposit Check and Charge Initialization Tests
 *
 * Trace:
 *   spec_id: SPEC-DEPOSIT-001
 *   task_id: TASK-004, TASK-011
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { checkDeposit } from './charge';
import type { HttpClient } from '../types';
import * as accountModule from './account';
import * as telegramModule from '../notify/telegram';
import { MIN_DEPOSIT_AMOUNT, USER_AGENT } from '../constants';

// Mock modules
vi.mock('./account');
vi.mock('../notify/telegram');

describe('checkDeposit', () => {
  let mockClient: HttpClient;

  beforeEach(() => {
    // Mock process.env
    vi.stubEnv('TELEGRAM_BOT_TOKEN', 'test-token');
    vi.stubEnv('TELEGRAM_CHAT_ID', 'test-chat-id');

    // Reset all mocks
    vi.clearAllMocks();

    // Create mock HTTP client
    mockClient = {
      cookies: {},
      fetch: vi.fn(),
      getCookieHeader: vi.fn(() => ''),
      clearCookies: vi.fn(),
    } as unknown as HttpClient;
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  describe('TEST-DEPOSIT-001: Allow purchase when balance is sufficient', () => {
    it('should return true when balance is exactly MIN_DEPOSIT_AMOUNT', async () => {
      // Given: Balance is exactly 30,000 KRW
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: MIN_DEPOSIT_AMOUNT,
        currentRound: 1200,
      });

      // When: Deposit check is performed
      const result = await checkDeposit(mockClient);

      // Then: Function returns true (proceed with purchase)
      expect(result).toBe(true);

      // And: No charge page initialization
      expect(mockClient.fetch).not.toHaveBeenCalledWith(
        expect.stringContaining('kbank.do'),
        expect.any(Object)
      );

      // And: No Telegram notification sent
      expect(telegramModule.sendNotification).not.toHaveBeenCalled();
    });

    it('should return true when balance is greater than MIN_DEPOSIT_AMOUNT', async () => {
      // Given: Balance is 50,000 KRW (> 30,000)
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 50000,
        currentRound: 1200,
      });

      // When: Deposit check is performed
      const result = await checkDeposit(mockClient);

      // Then: Function returns true
      expect(result).toBe(true);

      // And: No charge initialization
      expect(mockClient.fetch).not.toHaveBeenCalledWith(
        expect.stringContaining('kbank.do'),
        expect.any(Object)
      );

      // And: No notification sent
      expect(telegramModule.sendNotification).not.toHaveBeenCalled();
    });
  });

  describe('TEST-DEPOSIT-002: Block purchase when balance is insufficient', () => {
    it('should return false when balance is less than MIN_DEPOSIT_AMOUNT', async () => {
      // Given: Balance is 4,000 KRW (< 5,000)
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 4000,
        currentRound: 1200,
      });

      // Mock successful charge page initialization
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      const result = await checkDeposit(mockClient);

      // Then: Function returns false (block purchase)
      expect(result).toBe(false);
    });

    it('should return false when balance is zero', async () => {
      // Given: Balance is 0 KRW
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 0,
        currentRound: 1200,
      });

      // Mock successful charge page initialization
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      const result = await checkDeposit(mockClient);

      // Then: Function returns false
      expect(result).toBe(false);
    });
  });

  describe('TEST-DEPOSIT-003: Initialize charge page correctly', () => {
    it('should make GET request to K-Bank charge initialization endpoint', async () => {
      // Given: Insufficient balance
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 3000,
        currentRound: 1200,
      });

      // Mock successful charge page access
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html><body>K-Bank Virtual Account</body></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      await checkDeposit(mockClient);

      // Then: GET request to K-Bank charge endpoint
      expect(mockClient.fetch).toHaveBeenCalledWith(
        'https://www.dhlottery.co.kr/kbank.do?method=kbankProcess&PayMethod=VBANK&VBankAccountName=%EB%8F%99%ED%96%89%EB%B3%B5%EA%B6%8C&LicenseKey=&VBankExpDate=&GoodsAmt=50000',
        {
          method: 'GET',
          headers: {
            'User-Agent': USER_AGENT,
          },
        }
      );
    });

    it('should access charge page but not execute payment', async () => {
      // Given: Insufficient balance
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 2000,
        currentRound: 1200,
      });

      // Mock charge page response (not payment confirmation)
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Charge initialization happens
      await checkDeposit(mockClient);

      // Then: Only GET request (no POST for payment execution)
      expect(mockClient.fetch).toHaveBeenCalledTimes(1);
      expect(mockClient.fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ method: 'GET' })
      );
    });

    it('should handle charge initialization failure gracefully', async () => {
      // Given: Insufficient balance and charge page fails
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 3000,
        currentRound: 1200,
      });

      // Mock charge page error
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 500,
        statusText: 'Internal Server Error',
        headers: new Headers(),
        text: async () => '',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      const result = await checkDeposit(mockClient);

      // Then: Still returns false (fail-safe)
      expect(result).toBe(false);

      // And: Error notification is sent
      expect(telegramModule.sendNotification).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'error',
          title: expect.stringContaining('Charge'),
        }),
      );
    });
  });

  describe('TEST-DEPOSIT-004: Notify user when balance is low', () => {
    it('should send warning notification with current balance when insufficient', async () => {
      // Given: Balance is 4,000 KRW (insufficient)
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 4000,
        currentRound: 1200,
      });

      // Mock successful charge initialization
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      await checkDeposit(mockClient);

      // Then: Telegram notification sent with 'warning' type
      expect(telegramModule.sendNotification).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'warning',
        }),
      );

      // And: Message includes current balance
      const notificationCall = vi.mocked(telegramModule.sendNotification).mock.calls[0];
      const payload = notificationCall[0];
      expect(payload.details).toHaveProperty('currentBalance', '4,000원');
    });

    it('should request manual deposit in notification message', async () => {
      // Given: Insufficient balance
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 2000,
        currentRound: 1200,
      });

      // Mock successful charge initialization
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      await checkDeposit(mockClient);

      // Then: Notification message requests manual deposit
      const notificationCall = vi.mocked(telegramModule.sendNotification).mock.calls[0];
      const payload = notificationCall[0];
      expect(payload.message).toContain('입금');
    });

    it('should include minimum required amount in notification', async () => {
      // Given: Insufficient balance
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: 4000,
        currentRound: 1200,
      });

      // Mock successful charge initialization
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      await checkDeposit(mockClient);

      // Then: Notification includes minimum threshold
      const notificationCall = vi.mocked(telegramModule.sendNotification).mock.calls[0];
      const payload = notificationCall[0];
      expect(payload.details).toHaveProperty('minimumRequired', '5,000원');
    });
  });

  describe('TEST-DEPOSIT-005: Use correct minimum threshold', () => {
    it('should use MIN_DEPOSIT_AMOUNT constant (30,000 KRW)', async () => {
      // Given: Balance is 1 won below threshold
      const balanceJustBelow = MIN_DEPOSIT_AMOUNT - 1;
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: balanceJustBelow,
        currentRound: 1200,
      });

      // Mock successful charge initialization
      vi.mocked(mockClient.fetch).mockResolvedValue({
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        text: async () => '<html></html>',
        json: async <T>() => ({}) as T,
      });

      // When: Deposit check is performed
      const result = await checkDeposit(mockClient);

      // Then: Should block purchase (threshold not met)
      expect(result).toBe(false);
    });

    it('should perform exact comparison (not approximate)', async () => {
      // Given: Balance is exactly MIN_DEPOSIT_AMOUNT
      vi.mocked(accountModule.getAccountInfo).mockResolvedValue({
        balance: MIN_DEPOSIT_AMOUNT,
        currentRound: 1200,
      });

      // When: Deposit check is performed
      const result = await checkDeposit(mockClient);

      // Then: Should allow purchase (exact match)
      expect(result).toBe(true);
    });

    it('should verify threshold value is 5,000', () => {
      // Then: Constant value is exactly 5,000 (same as purchase cost)
      expect(MIN_DEPOSIT_AMOUNT).toBe(5000);
    });
  });

  describe('Error handling', () => {
    it('should throw error when account info fetch fails', async () => {
      // Given: Account info fetch fails
      const error = new Error('Network error');
      vi.mocked(accountModule.getAccountInfo).mockRejectedValue(error);

      // When/Then: Should throw error (fail-safe)
      await expect(checkDeposit(mockClient)).rejects.toThrow('Network error');

      // And: No notification sent (error propagates)
      expect(telegramModule.sendNotification).not.toHaveBeenCalled();
    });
  });
});
