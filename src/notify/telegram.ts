/**
 * Telegram Notification Service
 *
 * Trace:
 *   spec_id: SPEC-TELEGRAM-001, SPEC-GHACTION-001
 *   task_id: TASK-007, TASK-011, TASK-GHACTION-001
 */

import type { NotificationPayload, TelegramMessage } from '../types';
import { getEnv } from '../utils/env';
import { logger } from '../utils/logger';

// Retried on transient errors (network, server-side, rate-limit).
// Permanent client errors (4xx outside this set) are not retried.
const RETRY_STATUSES = new Set([408, 425, 429, 500, 502, 503, 504]);
const RETRY_DELAYS = [500, 1500]; // ms between attempts; length + 1 = total attempts

async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Escape Telegram Markdown v1 special characters
 * Escapes \ first to avoid double-escaping, then _ * ` [
 */
function escapeTelegramMarkdown(text: string): string {
  return text.replace(/\\/g, '\\\\').replace(/([_*`[])/g, '\\$1');
}

/**
 * Get emoji based on notification type
 */
function getTypeEmoji(type: NotificationPayload['type']): string {
  switch (type) {
    case 'success':
      return '✅';
    case 'warning':
      return '⚠️';
    case 'error':
      return '❌';
  }
}

/**
 * Format notification payload into Telegram message text
 */
function formatMessage(payload: NotificationPayload): string {
  const emoji = getTypeEmoji(payload.type);
  const lines: string[] = [];

  // Title with emoji
  lines.push(`${emoji} **${escapeTelegramMarkdown(payload.title)}**`);
  lines.push('');

  // Main message
  lines.push(escapeTelegramMarkdown(payload.message));

  // Add details if present
  if (payload.details && Object.keys(payload.details).length > 0) {
    lines.push('');
    for (const [key, value] of Object.entries(payload.details)) {
      // Format key in title case
      const formattedKey = key
        .replace(/([A-Z])/g, ' $1')
        .replace(/^./, (str) => str.toUpperCase())
        .trim();

      lines.push(
        `- ${escapeTelegramMarkdown(formattedKey)}: ${escapeTelegramMarkdown(String(value))}`
      );
    }
  }

  return lines.join('\n');
}

// Returns true on success, false after exhausting retries or a permanent error.
async function sendTelegramMessage(
  text: string,
  failureEvent: 'telegram_send_failed' | 'telegram_combined_send_failed'
): Promise<boolean> {
  const maxAttempts = RETRY_DELAYS.length + 1;

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      const botToken = getEnv('TELEGRAM_BOT_TOKEN');
      const chatId = getEnv('TELEGRAM_CHAT_ID');

      const message: TelegramMessage = {
        chat_id: chatId,
        text,
        parse_mode: 'Markdown',
      };

      const url = `https://api.telegram.org/bot${botToken}/sendMessage`;

      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(message),
      });

      if (response.ok) return true;

      if (RETRY_STATUSES.has(response.status)) {
        if (attempt < RETRY_DELAYS.length) {
          logger.warn('Telegram API error, retrying', {
            event: 'telegram_retry_attempt',
            attempt: attempt + 1,
            status: response.status,
          });
          await sleep(RETRY_DELAYS[attempt]);
          continue;
        }
        logger.error('Telegram notification failed after retries', {
          event: 'telegram_final_failure',
          failureEvent,
          attempts: maxAttempts,
          status: response.status,
        });
        return false;
      }

      // Permanent client error — log and stop
      const errorData = await response.json();
      logger.error('Telegram API error', {
        event: 'telegram_api_error',
        status: response.status,
        statusText: response.statusText,
        errorData,
      });
      return false;
    } catch (error) {
      if (attempt < RETRY_DELAYS.length) {
        logger.warn('Telegram send failed, retrying', {
          event: 'telegram_retry_attempt',
          attempt: attempt + 1,
          error: error instanceof Error ? error.message : String(error),
        });
        await sleep(RETRY_DELAYS[attempt]);
        continue;
      }
      logger.error('Failed to send Telegram notification', {
        event: failureEvent,
        error: error instanceof Error ? error.message : String(error),
      });
      return false;
    }
  }
  return false;
}

/**
 * Format and send multiple notification payloads as a single Telegram message.
 * Each payload is separated by a `---` divider.
 * Returns false if the send fails after all retries; true if sent or empty.
 */
export async function sendCombinedNotification(payloads: NotificationPayload[]): Promise<boolean> {
  if (payloads.length === 0) return true;

  const combinedText = payloads.map((p) => formatMessage(p)).join('\n\n---\n\n');
  return sendTelegramMessage(combinedText, 'telegram_combined_send_failed');
}

/**
 * Send notification to Telegram.
 * Returns false if the send fails after all retries; true on success.
 */
export async function sendNotification(payload: NotificationPayload): Promise<boolean> {
  const text = formatMessage(payload);
  return sendTelegramMessage(text, 'telegram_send_failed');
}
