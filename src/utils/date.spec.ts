/**
 * Date Utilities Tests
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001
 *   task_id: TASK-010
 */

import { describe, expect, it } from 'vitest';

import { calculatePreviousWeekRangeKst } from './date';

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

