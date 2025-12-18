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
export type { HttpClient, HttpRequestOptions, HttpResponse, CookieStore } from './http.types';
export type {
  PurchaseEnv,
  PurchaseReadyResponse,
  GameSelection,
  PurchaseResult,
  PurchaseSuccess,
  PurchaseFailure,
  PurchaseOutcome,
} from './purchase.types';
export type {
  NotificationType,
  NotificationPayload,
  TelegramEnv,
  TelegramMessage,
} from './notification.types';
export type { WinningResult } from './winning.types';

// Re-export for composed type
import type { AuthEnv } from './auth.types';
import type { TelegramEnv } from './notification.types';

/**
 * Worker environment combining all required env vars
 */
export type WorkerEnv = AuthEnv & TelegramEnv;
