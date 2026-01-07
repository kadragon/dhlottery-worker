/**
 * Winning Check Module
 *
 * Trace:
 *   spec_id: SPEC-WINNING-001, SPEC-REFACTOR-P2-LOG-001, SPEC-GHACTION-001
 *   task_id: TASK-006, TASK-010, TASK-011, TASK-REFACTOR-P2-002, TASK-GHACTION-001
 */

import { USER_AGENT } from '../constants';
import { sendNotification } from '../notify/telegram';
import type { HttpClient, WinningResult } from '../types';
import type { PreviousWeekRange } from '../utils/date';
import { calculatePreviousWeekRangeKst } from '../utils/date';
import { DHLotteryError } from '../utils/errors';
import { formatKoreanNumber } from '../utils/format';
import { logger } from '../utils/logger';

const WINNING_LIST_URL = 'https://www.dhlottery.co.kr/myPage.do?method=lottoBuyList';

/**
 * Winning result parsing regex patterns
 *
 * Matches various HTML elements in the lottery winning results page:
 * - Table rows: <tr>...</tr>
 * - Table cells: <td>...</td>
 * - JavaScript function calls: detailPop('arg1', 'arg2', 'roundNumber')
 * - Match counts: "N개" (N matches)
 * - Prize amounts: numeric digits
 *
 * Trace: spec_id: SPEC-REFACTOR-P2-REGEX-001, task_id: TASK-REFACTOR-P2-001
 */
export const WINNING_PATTERNS = {
  // Extracts table rows from HTML
  rowExtraction: /<tr\b[\s\S]*?<\/tr>/gi,
  // Extracts table cell content
  tableCell: /<td\b[^>]*>([\s\S]*?)<\/td>/gi,
  // Extracts round number from detailPop function call
  detailPopCall: /detailPop\(\s*'[^']*'\s*,\s*'[^']*'\s*,\s*'(\d+)'\s*\)/i,
  // Extracts first digit sequence (rank number)
  digitExtraction: /(\d+)/,
  // Extracts match count with Korean counter "N개"
  matchCount: /(\d+)\s*개/,
} as const;

export type { PreviousWeekRange };

/**
 * Calculate previous week range (Monday 00:00 KST to Sunday 23:59:59.999 KST)
 */
export function calculatePreviousWeekRange(now: Date = new Date()): PreviousWeekRange {
  try {
    return calculatePreviousWeekRangeKst(now);
  } catch (error) {
    if (error instanceof DHLotteryError) throw error;
    throw new DHLotteryError('Invalid date range computed', 'WINNING_INVALID_DATE_RANGE');
  }
}

function stripHtmlTags(input: string): string {
  return input
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

/**
 * Extract text from table cells in a row
 *
 * Uses String.matchAll to avoid global regex lastIndex bugs.
 * The shared global regex WINNING_PATTERNS.tableCell would otherwise
 * retain lastIndex across calls, causing rows after the first to be
 * partially or completely skipped.
 *
 * Trace: spec_id: SPEC-REFACTOR-P2-REGEX-001, task_id: TASK-REFACTOR-P2-001
 */
function extractTdTexts(rowHtml: string): string[] {
  const matches = rowHtml.matchAll(WINNING_PATTERNS.tableCell);
  return Array.from(matches, (m) => stripHtmlTags(m[1] ?? ''));
}

/**
 * Parse winning results from lottoBuyList HTML.
 *
 * Expected row structure (indices):
 * 0: buyDate, 1: gameName, 2: detail/info, 3: count, 4: result, 5: prize, 6: drawDate
 *
 * Trace: spec_id: SPEC-REFACTOR-P2-REGEX-001, task_id: TASK-REFACTOR-P2-001
 */
export function parseWinningResultsFromHtml(html: string): WinningResult[] {
  const rows = html.match(WINNING_PATTERNS.rowExtraction) ?? [];
  const results: WinningResult[] = [];

  for (const row of rows) {
    // Skip header rows that usually contain <th>
    if (/<th\b/i.test(row)) continue;

    const cells = extractTdTexts(row);
    if (cells.length < 6) continue;

    // Round number is typically present in detailPop third arg (issueNo).
    const issueMatch = row.match(WINNING_PATTERNS.detailPopCall);
    const roundNumberStr = issueMatch?.[1];
    const roundNumber = roundNumberStr ? Number.parseInt(roundNumberStr, 10) : Number.NaN;

    // Result cell: first number is treated as rank (robust to encoding issues).
    const resultCell = cells[4] ?? '';
    const rankMatch = resultCell.match(WINNING_PATTERNS.digitExtraction);
    const rankStr = rankMatch?.[1];
    const rank = rankStr ? Number.parseInt(rankStr, 10) : Number.NaN;

    const matchCountMatch = resultCell.match(WINNING_PATTERNS.matchCount);
    const matchCountStr = matchCountMatch?.[1];
    const matchCount = matchCountStr ? Number.parseInt(matchCountStr, 10) : undefined;

    // Prize cell: digits only
    const prizeCell = cells[5] ?? '';
    const prizeDigits = prizeCell.replace(/[^\d]/g, '');
    const prizeAmount = prizeDigits ? Number.parseInt(prizeDigits, 10) : Number.NaN;

    if (!Number.isFinite(roundNumber) || !Number.isFinite(rank) || !Number.isFinite(prizeAmount)) {
      continue;
    }
    if (rank < 1) continue;
    if (prizeAmount < 0) continue;

    results.push({
      roundNumber,
      rank,
      prizeAmount,
      matchCount,
    });
  }

  return results;
}

/**
 * Filter for rank 1 (jackpot) wins only.
 */
export function filterJackpotWins(results: WinningResult[]): WinningResult[] {
  return results.filter((r) => r.rank === 1);
}

/**
 * Check winning results for the previous week and notify jackpot wins.
 *
 * Non-fatal by design: network/parsing errors return empty array.
 * Critical: invalid date range throws.
 */
export async function checkWinning(
  client: HttpClient,
  now: Date = new Date()
): Promise<WinningResult[]> {
  const { startDate, endDate } = calculatePreviousWeekRange(now);

  try {
    const url = new URL(WINNING_LIST_URL);
    url.searchParams.set('searchStartDate', startDate);
    url.searchParams.set('searchEndDate', endDate);
    url.searchParams.set('nowPage', '1');

    const response = await client.fetch(url.toString(), {
      headers: {
        'User-Agent': USER_AGENT,
      },
    });
    // Accept 200 OK or 3xx redirect
    if (response.status !== 200 && (response.status < 300 || response.status >= 400)) {
      logger.error('Winning fetch failed', {
        event: 'winning_fetch_failed',
        status: response.status,
      });
      return [];
    }

    // Get HTML content with euc-kr encoding (server sends euc-kr)
    const html = await response.text('euc-kr');
    const parsed = parseWinningResultsFromHtml(html);
    const jackpotWins = filterJackpotWins(parsed);

    if (jackpotWins.length === 0) return [];

    for (const win of jackpotWins) {
      await sendNotification({
        type: 'success',
        title: 'Lottery Jackpot Win!',
        message: `${win.roundNumber}회차 로또 ${win.rank}등 당첨을 확인했습니다.`,
        details: {
          roundNumber: win.roundNumber,
          rank: win.rank,
          prizeAmount: win.prizeAmount,
          prizeAmountKrw: `${formatKoreanNumber(win.prizeAmount)}원`,
          matchCount: win.matchCount,
          period: `${startDate} ~ ${endDate}`,
        },
      });
    }

    return jackpotWins;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    logger.error('Winning check failed (non-fatal)', {
      event: 'winning_check_failed',
      error: message,
    });
    return [];
  }
}
