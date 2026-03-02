import type { NotificationPayload } from '../types';
import type { NotificationCollector } from './notification-collector';
import { sendNotification } from './telegram';

export async function notify(
  payload: NotificationPayload,
  collector?: NotificationCollector
): Promise<void> {
  if (collector) {
    collector.add(payload);
  } else {
    await sendNotification(payload);
  }
}
