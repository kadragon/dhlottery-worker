/**
 * Environment Variable Utilities Tests
 *
 * Trace:
 *   spec_id: SPEC-GHACTION-001
 *   task_id: TASK-GHACTION-001
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getEnv, validateEnv } from './env';

describe('Environment Variable Utilities', () => {
  beforeEach(() => {
    // Clear all environment variables before each test
    vi.unstubAllEnvs();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  describe('getEnv()', () => {
    it('should return the value when environment variable exists', () => {
      vi.stubEnv('USER_ID', 'test-user-id');

      const result = getEnv('USER_ID');

      expect(result).toBe('test-user-id');
    });

    it('should return correct values for all required environment variables', () => {
      vi.stubEnv('USER_ID', 'user123');
      vi.stubEnv('PASSWORD', 'pass456');
      vi.stubEnv('TELEGRAM_BOT_TOKEN', 'bot-token-789');
      vi.stubEnv('TELEGRAM_CHAT_ID', 'chat-id-012');

      expect(getEnv('USER_ID')).toBe('user123');
      expect(getEnv('PASSWORD')).toBe('pass456');
      expect(getEnv('TELEGRAM_BOT_TOKEN')).toBe('bot-token-789');
      expect(getEnv('TELEGRAM_CHAT_ID')).toBe('chat-id-012');
    });

    it('should throw error when environment variable is missing', () => {
      // USER_ID is not set

      expect(() => getEnv('USER_ID')).toThrowError(
        'Missing required environment variable: USER_ID'
      );
    });

    it('should throw error when environment variable is empty string', () => {
      vi.stubEnv('PASSWORD', '');

      expect(() => getEnv('PASSWORD')).toThrowError(
        'Missing required environment variable: PASSWORD'
      );
    });

    it('should throw error with correct variable name in message', () => {
      expect(() => getEnv('TELEGRAM_BOT_TOKEN')).toThrowError(/TELEGRAM_BOT_TOKEN/);
      expect(() => getEnv('TELEGRAM_CHAT_ID')).toThrowError(/TELEGRAM_CHAT_ID/);
    });
  });

  describe('validateEnv()', () => {
    it('should succeed when all required environment variables are set', () => {
      vi.stubEnv('USER_ID', 'user123');
      vi.stubEnv('PASSWORD', 'pass456');
      vi.stubEnv('TELEGRAM_BOT_TOKEN', 'bot-token-789');
      vi.stubEnv('TELEGRAM_CHAT_ID', 'chat-id-012');

      expect(() => validateEnv()).not.toThrow();
    });

    it('should throw error when one required variable is missing', () => {
      vi.stubEnv('USER_ID', 'user123');
      vi.stubEnv('PASSWORD', 'pass456');
      vi.stubEnv('TELEGRAM_BOT_TOKEN', 'bot-token-789');
      // TELEGRAM_CHAT_ID is missing

      expect(() => validateEnv()).toThrowError(
        'Missing required environment variables: TELEGRAM_CHAT_ID'
      );
    });

    it('should throw error listing all missing variables', () => {
      vi.stubEnv('USER_ID', 'user123');
      // PASSWORD, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID are missing

      expect(() => validateEnv()).toThrowError(
        'Missing required environment variables: PASSWORD, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID'
      );
    });

    it('should throw error when all required variables are missing', () => {
      // No environment variables set

      expect(() => validateEnv()).toThrowError(
        'Missing required environment variables: USER_ID, PASSWORD, TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID'
      );
    });

    it('should treat empty string as missing variable', () => {
      vi.stubEnv('USER_ID', 'user123');
      vi.stubEnv('PASSWORD', ''); // Empty string
      vi.stubEnv('TELEGRAM_BOT_TOKEN', 'bot-token-789');
      vi.stubEnv('TELEGRAM_CHAT_ID', 'chat-id-012');

      expect(() => validateEnv()).toThrowError(/PASSWORD/);
    });

    it('should throw error with correct message format', () => {
      vi.stubEnv('USER_ID', 'user123');
      vi.stubEnv('PASSWORD', 'pass456');
      // TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are missing

      expect(() => validateEnv()).toThrowError(/^Missing required environment variables: /);
    });
  });

  describe('Edge cases', () => {
    it('should handle whitespace-only values as valid', () => {
      // Note: Current implementation treats whitespace as valid
      // This test documents the current behavior
      vi.stubEnv('USER_ID', '   ');

      const result = getEnv('USER_ID');

      expect(result).toBe('   ');
    });

    it('should be case-sensitive for variable names', () => {
      vi.stubEnv('user_id', 'lowercase');
      vi.stubEnv('USER_ID', 'uppercase');

      const result = getEnv('USER_ID');

      expect(result).toBe('uppercase');
    });
  });
});
