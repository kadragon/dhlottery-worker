/**
 * Telegram Notification Service
 *
 * Trace:
 *   spec_id: SPEC-TELEGRAM-001
 *   task_id: TASK-007
 */

import type {
  NotificationPayload,
  TelegramEnv,
  TelegramMessage,
} from "../types/notification.types";

/**
 * Get emoji based on notification type
 */
function getTypeEmoji(type: NotificationPayload["type"]): string {
  switch (type) {
    case "success":
      return "✅";
    case "warning":
      return "⚠️";
    case "error":
      return "❌";
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
  lines.push("");

  // Main message
  lines.push(payload.message);

  // Add details if present
  if (payload.details && Object.keys(payload.details).length > 0) {
    lines.push("");
    for (const [key, value] of Object.entries(payload.details)) {
      // Format key in title case
      const formattedKey = key
        .replace(/([A-Z])/g, " $1")
        .replace(/^./, (str) => str.toUpperCase())
        .trim();

      lines.push(`- ${formattedKey}: ${value}`);
    }
  }

  return lines.join("\n");
}

/**
 * Send notification to Telegram
 *
 * @param payload - Notification payload with type, title, message, and optional details
 * @param env - Environment containing Telegram credentials
 */
export async function sendNotification(
  payload: NotificationPayload,
  env: TelegramEnv,
): Promise<void> {
  try {
    // Format message text
    const text = formatMessage(payload);

    // Prepare Telegram API message
    const message: TelegramMessage = {
      chat_id: env.TELEGRAM_CHAT_ID,
      text,
      parse_mode: "Markdown",
    };

    // Build API URL
    const url = `https://api.telegram.org/bot${env.TELEGRAM_BOT_TOKEN}/sendMessage`;

    // Send request to Telegram API
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(message),
    });

    // Check response
    if (!response.ok) {
      const errorData = await response.json();
      console.error(
        `Telegram API error: ${response.status} ${response.statusText}`,
        errorData,
      );
      return; // Do not retry, just log
    }

    // Success - no need to process response
  } catch (error) {
    // Log network errors or other failures
    console.error("Failed to send Telegram notification:", error);
    // Do not throw - allow main execution to continue
  }
}
