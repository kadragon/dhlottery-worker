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

async function sendTelegramMessage(
  text: string,
  failureEvent: 'telegram_send_failed' | 'telegram_combined_send_failed'
): Promise<void> {
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

    if (!response.ok) {
      const errorData = await response.json();
      logger.error('Telegram API error', {
        event: 'telegram_api_error',
        status: response.status,
        statusText: response.statusText,
        errorData,
      });
    }
  } catch (error) {
    logger.error('Failed to send Telegram notification', {
      event: failureEvent,
      error: error instanceof Error ? error.message : String(error),
    });
  }
}

/**
 * Format and send multiple notification payloads as a single Telegram message.
 * Each payload is separated by a `---` divider.
 * No-op if the payload list is empty.
 */
export async function sendCombinedNotification(payloads: NotificationPayload[]): Promise<void> {
  if (payloads.length === 0) return;

  const combinedText = payloads.map((p) => formatMessage(p)).join('\n\n---\n\n');
  await sendTelegramMessage(combinedText, 'telegram_combined_send_failed');
}

/**
 * Send notification to Telegram
 *
 * @param payload - Notification payload with type, title, message, and optional details
 */
export async function sendNotification(payload: NotificationPayload): Promise<void> {
  const text = formatMessage(payload);
  await sendTelegramMessage(text, 'telegram_send_failed');
}
