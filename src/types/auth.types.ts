/**
 * Authentication Types
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001
 *   task_id: TASK-002
 */

/**
 * User credentials from Cloudflare Workers Secrets
 */
export interface Credentials {
  userId: string;
  password: string;
}

/**
 * Environment variables containing secrets
 */
export interface AuthEnv {
  USER_ID: string;
  PASSWORD: string;
}

/**
 * Login request payload
 */
export interface LoginPayload {
  userId: string;
  password: string;
  // Additional fields may be required by DHLottery
  [key: string]: string;
}

/**
 * Login response structure
 */
export interface LoginResponse {
  success: boolean;
  message?: string;
  // DHLottery may return additional fields
  [key: string]: unknown;
}
