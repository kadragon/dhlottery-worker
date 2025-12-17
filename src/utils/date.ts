/**
 * Date Utilities (KST-aware)
 *
 * Trace:
 *   spec_id: SPEC-UTILS-001
 *   task_id: TASK-010
 */

import { DHLotteryError } from './errors';

const KST_OFFSET_MS = 9 * 60 * 60 * 1000;
const ONE_DAY_MS = 24 * 60 * 60 * 1000;

export interface PreviousWeekRange {
  start: Date;
  end: Date;
  startDate: string; // YYYY-MM-DD in KST
  endDate: string; // YYYY-MM-DD in KST
}

export function formatKstDateYyyyMmDd(date: Date): string {
  const kst = new Date(date.getTime() + KST_OFFSET_MS);
  const year = kst.getUTCFullYear();
  const month = String(kst.getUTCMonth() + 1).padStart(2, '0');
  const day = String(kst.getUTCDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Calculate previous week range (Monday 00:00 KST to Sunday 23:59:59.999 KST)
 */
export function calculatePreviousWeekRangeKst(now: Date = new Date()): PreviousWeekRange {
  if (Number.isNaN(now.getTime())) {
    throw new DHLotteryError('Invalid now date', 'WINNING_INVALID_DATE_RANGE');
  }

  // Shift into KST so that UTC-based getters operate in KST.
  const kstNow = new Date(now.getTime() + KST_OFFSET_MS);
  const kstYear = kstNow.getUTCFullYear();
  const kstMonth = kstNow.getUTCMonth();
  const kstDate = kstNow.getUTCDate();
  const kstDayOfWeek = kstNow.getUTCDay(); // 0=Sun ... 6=Sat, in KST context

  // Days since Monday (Mon=0..Sun=6)
  const daysSinceMonday = (kstDayOfWeek + 6) % 7;

  // Start of current week in KST (Mon 00:00), represented as a UTC timestamp.
  const startOfCurrentWeekUtcMs =
    Date.UTC(kstYear, kstMonth, kstDate, 0, 0, 0, 0) - KST_OFFSET_MS - daysSinceMonday * ONE_DAY_MS;

  const startOfPreviousWeek = new Date(startOfCurrentWeekUtcMs - 7 * ONE_DAY_MS);
  const endOfPreviousWeek = new Date(startOfPreviousWeek.getTime() + 7 * ONE_DAY_MS - 1);

  if (startOfPreviousWeek.getTime() > endOfPreviousWeek.getTime()) {
    throw new DHLotteryError('Invalid date range computed', 'WINNING_INVALID_DATE_RANGE');
  }

  return {
    start: startOfPreviousWeek,
    end: endOfPreviousWeek,
    startDate: formatKstDateYyyyMmDd(startOfPreviousWeek),
    endDate: formatKstDateYyyyMmDd(endOfPreviousWeek),
  };
}
