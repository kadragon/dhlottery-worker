/**
 * Date Utilities Tests
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001
 *   task_id: TASK-010
 */

import { describe, expect, it } from 'vitest';

import { calculatePreviousWeekRangeKst, getNextSaturdayKst } from './date';

describe('TEST-UTILS-001: Previous week range in KST', () => {
  it('should calculate previous Monday to Sunday in KST', () => {
    // Monday 10:00 KST
    const nowKstMonday = new Date('2025-12-15T10:00:00+09:00');

    const { startDate, endDate } = calculatePreviousWeekRangeKst(nowKstMonday);

    // Previous week: 2025-12-08 (Mon) ~ 2025-12-14 (Sun) in KST
    expect(startDate).toBe('2025-12-08');
    expect(endDate).toBe('2025-12-14');
  });

  it('should throw for invalid Date input', () => {
    expect(() => calculatePreviousWeekRangeKst(new Date('invalid'))).toThrowError();
  });
});

describe('TEST-UTILS-002: Next Saturday calculation in KST', () => {
  it('should return this Saturday when today is Monday', () => {
    // Monday Dec 15, 2025 10:00 KST
    const monday = new Date('2025-12-15T10:00:00+09:00');
    const result = getNextSaturdayKst(monday);
    // This Saturday: 2025-12-20
    expect(result).toBe('2025-12-20');
  });

  it('should return this Saturday when today is Saturday', () => {
    // Saturday Dec 20, 2025 10:00 KST
    const saturday = new Date('2025-12-20T10:00:00+09:00');
    const result = getNextSaturdayKst(saturday);
    // Today (Saturday): 2025-12-20
    expect(result).toBe('2025-12-20');
  });

  it('should return next Saturday when today is Sunday', () => {
    // Sunday Dec 21, 2025 10:00 KST
    const sunday = new Date('2025-12-21T10:00:00+09:00');
    const result = getNextSaturdayKst(sunday);
    // Next Saturday: 2025-12-27
    expect(result).toBe('2025-12-27');
  });

  it('should throw for invalid Date input', () => {
    expect(() => getNextSaturdayKst(new Date('invalid'))).toThrowError();
  });
});

