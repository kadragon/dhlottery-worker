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
  lines.push(`${emoji} **${payload.title}**`);
  lines.push('');

  // Main message
  lines.push(payload.message);

  // Add details if present
  if (payload.details && Object.keys(payload.details).length > 0) {
    lines.push('');
    for (const [key, value] of Object.entries(payload.details)) {
      // Format key in title case
      const formattedKey = key
        .replace(/([A-Z])/g, ' $1')
        .replace(/^./, (str) => str.toUpperCase())
        .trim();

      lines.push(`- ${formattedKey}: ${value}`);
    }
  }

  return lines.join('\n');
}

/**
 * Send notification to Telegram
 *
 * @param payload - Notification payload with type, title, message, and optional details
 */
export async function sendNotification(payload: NotificationPayload): Promise<void> {
  try {
    // Format message text
    const text = formatMessage(payload);

    // Get Telegram credentials from environment
    const botToken = getEnv('TELEGRAM_BOT_TOKEN');
    const chatId = getEnv('TELEGRAM_CHAT_ID');

    // Prepare Telegram API message
    const message: TelegramMessage = {
      chat_id: chatId,
      text,
      parse_mode: 'Markdown',
    };

    // Build API URL
    const url = `https://api.telegram.org/bot${botToken}/sendMessage`;

    // Send request to Telegram API
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(message),
    });

    // Check response
    if (!response.ok) {
      const errorData = await response.json();
      logger.error('Telegram API error', {
        event: 'telegram_api_error',
        status: response.status,
        statusText: response.statusText,
        errorData,
      });
      return; // Do not retry, just log
    }

    // Success - no need to process response
  } catch (error) {
    // Log network errors or other failures
    logger.error('Failed to send Telegram notification', {
      event: 'telegram_send_failed',
      error: error instanceof Error ? error.message : String(error),
    });
    // Do not throw - allow main execution to continue
  }
}
