import type { NotificationPayload } from '../types';

export class NotificationCollector {
  private payloads: NotificationPayload[] = [];

  add(payload: NotificationPayload): void {
    this.payloads.push(payload);
  }

  getPayloads(): NotificationPayload[] {
    return [...this.payloads];
  }

  isEmpty(): boolean {
    return this.payloads.length === 0;
  }
}
