/**
 * Notification Types
 *
 * Trace:
 *   spec_id: SPEC-TELEGRAM-001, SPEC-REFACTOR-P1-TYPE-001
 *   task_id: TASK-007, TASK-REFACTOR-P1-002
 */

/**
 * Notification type
 */
export type NotificationType = 'success' | 'warning' | 'error';

/**
 * Notification payload
 */
export interface NotificationPayload {
  type: NotificationType;
  title: string;
  message: string;
  details?: Record<string, string | number | boolean | null | undefined>;
}

/**
 * Telegram environment variables
 */
export interface TelegramEnv {
  TELEGRAM_BOT_TOKEN: string;
  TELEGRAM_CHAT_ID: string;
}

/**
 * Telegram API message request
 */
export interface TelegramMessage {
  chat_id: string;
  text: string;
  parse_mode: 'Markdown' | 'MarkdownV2';
}
