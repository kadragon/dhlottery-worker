import { describe, it, expect } from 'vitest';
import { NotificationCollector } from './notification-collector';
import type { NotificationPayload } from '../types';

describe('NotificationCollector', () => {
  it('should buffer payloads added via add()', () => {
    const collector = new NotificationCollector();
    const payload: NotificationPayload = {
      type: 'success',
      title: 'Test',
      message: 'Test message',
    };

    collector.add(payload);

    expect(collector.getPayloads()).toEqual([payload]);
  });

  it('should return true from isEmpty() when no payloads added', () => {
    const collector = new NotificationCollector();
    expect(collector.isEmpty()).toBe(true);
  });

  it('should return false from isEmpty() after adding a payload', () => {
    const collector = new NotificationCollector();
    collector.add({ type: 'success', title: 'Test', message: 'msg' });
    expect(collector.isEmpty()).toBe(false);
  });

  it('should return a snapshot that does not mutate internal state', () => {
    const collector = new NotificationCollector();
    const payload: NotificationPayload = {
      type: 'success',
      title: 'Test',
      message: 'Test message',
    };

    collector.add(payload);
    const snapshot = collector.getPayloads();
    snapshot.push({
      type: 'error',
      title: 'Extra',
      message: 'Should not affect collector',
    });

    expect(collector.getPayloads()).toEqual([payload]);
  });
});
