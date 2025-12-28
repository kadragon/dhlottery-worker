/**
 * Types Barrel Export
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001
 *   task_id: TASK-010
 */

export type { AccountInfo } from './account.types';
export type { AuthEnv, Credentials, LoginPayload, LoginResponse } from './auth.types';
export type { DepositEnv } from './deposit.types';
export type { CookieStore, HttpClient, HttpRequestOptions, HttpResponse } from './http.types';
export type {
  NotificationPayload,
  NotificationType,
  TelegramEnv,
  TelegramMessage,
} from './notification.types';
export type {
  GameSelection,
  PurchaseEnv,
  PurchaseFailure,
  PurchaseOutcome,
  PurchaseReadyResponse,
  PurchaseResult,
  PurchaseSuccess,
} from './purchase.types';
export type { WinningResult } from './winning.types';

// Re-export for composed type
import type { AuthEnv } from './auth.types';
import type { TelegramEnv } from './notification.types';

/**
 * Worker environment combining all required env vars
 */
export type WorkerEnv = AuthEnv & TelegramEnv;
