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

/**
 * Calculate next Saturday in KST
 * Used for lottery draw date (draws happen every Saturday 20:00 KST)
 *
 * @param now - Current date (defaults to now)
 * @returns YYYY-MM-DD format string for the next Saturday in KST
 */
export function getNextSaturdayKst(now: Date = new Date()): string {
  if (Number.isNaN(now.getTime())) {
    throw new DHLotteryError('Invalid date', 'INVALID_DATE');
  }

  // Convert to KST
  const kstNow = new Date(now.getTime() + KST_OFFSET_MS);
  const kstDayOfWeek = kstNow.getUTCDay(); // 0=Sun, 6=Sat

  // Calculate days until next Saturday (6)
  let daysUntilSaturday: number;
  if (kstDayOfWeek === 6) {
    // Today is Saturday - use today
    daysUntilSaturday = 0;
  } else if (kstDayOfWeek === 0) {
    // Today is Sunday - 6 days until next Saturday
    daysUntilSaturday = 6;
  } else {
    // Monday(1) to Friday(5) - calculate remaining days
    daysUntilSaturday = 6 - kstDayOfWeek;
  }

  const nextSaturday = new Date(now.getTime() + daysUntilSaturday * ONE_DAY_MS);
  return formatKstDateYyyyMmDd(nextSaturday);
}

/**
 * Normalize a date string to YYYY-MM-DD format.
 * Accepts both "YYYYMMDD" and "YYYY-MM-DD".
 */
function normalizeYmdDate(dateStr: string): string {
  return dateStr.includes('-')
    ? dateStr
    : `${dateStr.slice(0, 4)}-${dateStr.slice(4, 6)}-${dateStr.slice(6, 8)}`;
}

/**
 * Add days to a date string (YYYY-MM-DD or YYYYMMDD).
 * Returns YYYY-MM-DD format.
 */
export function addDaysToYmd(dateStr: string, days: number): string {
  const normalized = normalizeYmdDate(dateStr);
  const [year, month, day] = normalized.split('-').map((part) => Number(part));
  const utcDate = new Date(Date.UTC(year, month - 1, day));
  utcDate.setUTCDate(utcDate.getUTCDate() + days);
  const nextYear = utcDate.getUTCFullYear();
  const nextMonth = String(utcDate.getUTCMonth() + 1).padStart(2, '0');
  const nextDay = String(utcDate.getUTCDate()).padStart(2, '0');
  return `${nextYear}-${nextMonth}-${nextDay}`;
}

/**
 * Add years and days to a YYYY-MM-DD date string.
 * Returns YYYY-MM-DD format.
 */
export function addYearsAndDays(date: string, years: number, days: number): string {
  const [year, month, day] = date.split('-').map((part) => Number(part));
  const base = new Date(Date.UTC(year, month - 1, day));
  base.setUTCFullYear(base.getUTCFullYear() + years);
  base.setUTCDate(base.getUTCDate() + days);
  const resultYear = base.getUTCFullYear();
  const resultMonth = String(base.getUTCMonth() + 1).padStart(2, '0');
  const resultDay = String(base.getUTCDate()).padStart(2, '0');
  return `${resultYear}-${resultMonth}-${resultDay}`;
}
