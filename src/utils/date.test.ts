/**
 * Date Utilities Tests
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001
 *   task_id: TASK-010
 */

import { describe, expect, it } from 'vitest';

import { addDaysToYmd, addYearsAndDays, calculatePreviousWeekRangeKst, getNextSaturdayKst } from './date';

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

  it('should return correct Saturday when KST time is late Saturday (previously double-offset bug)', () => {
    // Saturday Dec 20, 2025 18:00 KST (= UTC 09:00)
    // With double KST offset this would incorrectly return Sunday
    const lateSaturday = new Date('2025-12-20T18:00:00+09:00');
    const result = getNextSaturdayKst(lateSaturday);
    expect(result).toBe('2025-12-20');
  });

  it('should throw for invalid Date input', () => {
    expect(() => getNextSaturdayKst(new Date('invalid'))).toThrowError();
  });
});

describe('addDaysToYmd', () => {
  it('should add days to YYYY-MM-DD format', () => {
    expect(addDaysToYmd('2025-12-28', 7)).toBe('2026-01-04');
  });

  it('should add days to YYYYMMDD format', () => {
    expect(addDaysToYmd('20251228', 7)).toBe('2026-01-04');
  });

  it('should handle month boundary', () => {
    expect(addDaysToYmd('2025-01-30', 2)).toBe('2025-02-01');
  });
});

describe('addYearsAndDays', () => {
  it('should add years and days to YYYY-MM-DD format', () => {
    expect(addYearsAndDays('2025-12-20', 1, 1)).toBe('2026-12-21');
  });

  it('should handle leap year boundary', () => {
    expect(addYearsAndDays('2024-02-29', 1, 0)).toBe('2025-03-01');
  });
});

