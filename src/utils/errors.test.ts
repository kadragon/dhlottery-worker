/**
 * Custom Error Classes Tests
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001, SPEC-REFACTOR-P1-ERROR-001
 *   task_id: TASK-002, TASK-REFACTOR-P1-003
 */

import { describe, it, expect } from 'vitest';
import {
  DHLotteryError,
  AuthenticationError,
  NetworkError,
  PurchaseError,
  wrapAuthError,
} from './errors';

describe('DHLottery Error Classes', () => {
  /**
   * Base Error Class Tests
   */
  describe('DHLotteryError', () => {
    it('should create error with message and code', () => {
      const error = new DHLotteryError('Test error', 'TEST_CODE');

      expect(error).toBeInstanceOf(Error);
      expect(error.message).toBe('Test error');
      expect(error.code).toBe('TEST_CODE');
      expect(error.name).toBe('DHLotteryError');
    });
  });

  /**
   * Authentication Error Tests
   */
  describe('AuthenticationError', () => {
    it('should extend DHLotteryError', () => {
      const error = new AuthenticationError('Auth failed');

      expect(error).toBeInstanceOf(DHLotteryError);
      expect(error).toBeInstanceOf(Error);
    });

    it('should have correct name', () => {
      const error = new AuthenticationError('Auth failed');

      expect(error.name).toBe('AuthenticationError');
    });

    it('should use default code if not provided', () => {
      const error = new AuthenticationError('Auth failed');

      expect(error.code).toBe('AUTH_ERROR');
    });

    it('should accept custom error code', () => {
      const error = new AuthenticationError('Invalid credentials', 'AUTH_INVALID_CREDENTIALS');

      expect(error.code).toBe('AUTH_INVALID_CREDENTIALS');
      expect(error.message).toBe('Invalid credentials');
    });
  });

  /**
   * Network Error Tests
   */
  describe('NetworkError', () => {
    it('should extend DHLotteryError', () => {
      const error = new NetworkError('Network failed');

      expect(error).toBeInstanceOf(DHLotteryError);
      expect(error).toBeInstanceOf(Error);
    });

    it('should have correct name', () => {
      const error = new NetworkError('Network failed');

      expect(error.name).toBe('NetworkError');
    });

    it('should use default code if not provided', () => {
      const error = new NetworkError('Network failed');

      expect(error.code).toBe('NETWORK_ERROR');
    });

    it('should accept custom error code', () => {
      const error = new NetworkError('Timeout', 'NETWORK_TIMEOUT');

      expect(error.code).toBe('NETWORK_TIMEOUT');
      expect(error.message).toBe('Timeout');
    });
  });

  /**
   * TEST-REFACTOR-P1-ERROR-001: PurchaseError extends DHLotteryError
   */
  describe('TEST-REFACTOR-P1-ERROR-001: PurchaseError inheritance', () => {
    it('should extend DHLotteryError', () => {
      const error = new PurchaseError('Purchase failed', 'TEST_CODE');

      expect(error).toBeInstanceOf(DHLotteryError);
      expect(error).toBeInstanceOf(Error);
    });
  });

  /**
   * TEST-REFACTOR-P1-ERROR-002: PurchaseError has correct name property
   */
  describe('TEST-REFACTOR-P1-ERROR-002: PurchaseError name', () => {
    it('should have name "PurchaseError"', () => {
      const error = new PurchaseError('Purchase failed');

      expect(error.name).toBe('PurchaseError');
    });
  });

  /**
   * TEST-REFACTOR-P1-ERROR-003: PurchaseError accepts optional error code
   */
  describe('TEST-REFACTOR-P1-ERROR-003: PurchaseError code handling', () => {
    it('should use default code if not provided', () => {
      const error = new PurchaseError('Purchase failed');

      expect(error.code).toBe('PURCHASE_ERROR');
    });

    it('should accept custom error code', () => {
      const error = new PurchaseError('Ready failed', 'PURCHASE_READY_FAILED');

      expect(error.code).toBe('PURCHASE_READY_FAILED');
    });

    it('should accept execution failed code', () => {
      const error = new PurchaseError('Execution failed', 'PURCHASE_EXECUTION_FAILED');

      expect(error.code).toBe('PURCHASE_EXECUTION_FAILED');
    });
  });

  /**
   * TEST-REFACTOR-P1-ERROR-004: PurchaseError preserves message
   */
  describe('TEST-REFACTOR-P1-ERROR-004: PurchaseError message', () => {
    it('should preserve error message', () => {
      const message = 'Purchase ready failed: 500';
      const error = new PurchaseError(message);

      expect(error.message).toBe(message);
    });

    it('should preserve message with custom code', () => {
      const message = 'Purchase execution failed: 400';
      const error = new PurchaseError(message, 'PURCHASE_EXECUTION_FAILED');

      expect(error.message).toBe(message);
      expect(error.code).toBe('PURCHASE_EXECUTION_FAILED');
    });
  });

  /**
   * TEST-REFACTOR-P0-AUTH-002: wrapAuthError behavior
   */
  describe('TEST-REFACTOR-P0-AUTH-002: wrapAuthError behavior', () => {
    it('should return existing AuthenticationError instance', () => {
      const error = new AuthenticationError('Auth failed', 'AUTH_INVALID_CREDENTIALS');

      const wrapped = wrapAuthError(error, 'Login');

      expect(wrapped).toBe(error);
    });

    it('should wrap Error with context and AUTH_NETWORK_ERROR code', () => {
      const error = new Error('boom');

      const wrapped = wrapAuthError(error, 'Login');

      expect(wrapped).toBeInstanceOf(AuthenticationError);
      expect(wrapped.message).toBe('Login failed: boom');
      expect(wrapped.code).toBe('AUTH_NETWORK_ERROR');
    });

    it('should handle non-Error values as Unknown error', () => {
      const wrapped = wrapAuthError('oops', 'Login');

      expect(wrapped.message).toBe('Login failed: Unknown error');
      expect(wrapped.code).toBe('AUTH_NETWORK_ERROR');
    });
  });
});
