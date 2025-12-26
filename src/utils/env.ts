/**
 * Environment Variable Utilities
 *
 * Trace:
 *   spec_id: SPEC-GHACTION-001
 *   task_id: TASK-GHACTION-001
 */

/**
 * Required environment variable keys
 */
const REQUIRED_ENV_KEYS = ['USER_ID', 'PASSWORD', 'TELEGRAM_BOT_TOKEN', 'TELEGRAM_CHAT_ID'] as const;

export type RequiredEnvKey = (typeof REQUIRED_ENV_KEYS)[number];

/**
 * Get a required environment variable
 * Throws an error if the variable is not set
 */
export function getEnv(key: RequiredEnvKey): string {
  const value = process.env[key];
  if (!value) {
    throw new Error(`Missing required environment variable: ${key}`);
  }
  return value;
}

/**
 * Validate all required environment variables
 * Throws an error with all missing variables if any are not set
 */
export function validateEnv(): void {
  const missing = REQUIRED_ENV_KEYS.filter((key) => !process.env[key]);

  if (missing.length > 0) {
    throw new Error(`Missing required environment variables: ${missing.join(', ')}`);
  }
}
