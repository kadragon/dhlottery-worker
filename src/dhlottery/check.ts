/**
 * Winning Check Module
 *
 * Trace:
 *   spec_id: SPEC-WINNING-001
 *   task_id: TASK-006, TASK-010, TASK-011
 */

import { sendNotification } from '../notify/telegram';
import type { HttpClient, TelegramEnv, WinningResult } from '../types';
import { calculatePreviousWeekRangeKst } from '../utils/date';
import type { PreviousWeekRange } from '../utils/date';
import { DHLotteryError } from '../utils/errors';

const WINNING_LIST_URL = 'https://www.dhlottery.co.kr/myPage.do?method=lottoBuyList';

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

function extractTdTexts(rowHtml: string): string[] {
  const cells: string[] = [];
  const tdRegex = /<td\b[^>]*>([\s\S]*?)<\/td>/gi;
  while (true) {
    const match = tdRegex.exec(rowHtml);
    if (!match) break;
    cells.push(stripHtmlTags(match[1] ?? ''));
  }
  return cells;
}

/**
 * Parse winning results from lottoBuyList HTML.
 *
 * Expected row structure (indices):
 * 0: buyDate, 1: gameName, 2: detail/info, 3: count, 4: result, 5: prize, 6: drawDate
 */
export function parseWinningResultsFromHtml(html: string): WinningResult[] {
  const rows = html.match(/<tr\b[\s\S]*?<\/tr>/gi) ?? [];
  const results: WinningResult[] = [];

  for (const row of rows) {
    // Skip header rows that usually contain <th>
    if (/<th\b/i.test(row)) continue;

    const cells = extractTdTexts(row);
    if (cells.length < 6) continue;

    // Round number is typically present in detailPop third arg (issueNo).
    const issueMatch = row.match(/detailPop\(\s*'[^']*'\s*,\s*'[^']*'\s*,\s*'(\d+)'\s*\)/i);
    const roundNumberStr = issueMatch?.[1];
    const roundNumber = roundNumberStr ? Number.parseInt(roundNumberStr, 10) : Number.NaN;

    // Result cell: first number is treated as rank (robust to encoding issues).
    const resultCell = cells[4] ?? '';
    const rankMatch = resultCell.match(/(\d+)/);
    const rankStr = rankMatch?.[1];
    const rank = rankStr ? Number.parseInt(rankStr, 10) : Number.NaN;

    const matchCountMatch = resultCell.match(/(\d+)\s*개/);
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

function formatKoreanNumber(amount: number): string {
  return amount.toLocaleString('ko-KR');
}

/**
 * Check winning results for the previous week and notify jackpot wins.
 *
 * Non-fatal by design: network/parsing errors return empty array.
 * Critical: invalid date range throws.
 */
export async function checkWinning(
  client: HttpClient,
  env: TelegramEnv,
  now: Date = new Date()
): Promise<WinningResult[]> {
  const { startDate, endDate } = calculatePreviousWeekRange(now);

  try {
    const url = new URL(WINNING_LIST_URL);
    url.searchParams.set('searchStartDate', startDate);
    url.searchParams.set('searchEndDate', endDate);
    url.searchParams.set('nowPage', '1');

    const response = await client.fetch(url.toString());
    if (response.status !== 200) {
      console.error(`Winning fetch failed: HTTP ${response.status}`);
      return [];
    }

    const html = await response.text();
    const parsed = parseWinningResultsFromHtml(html);
    const jackpotWins = filterJackpotWins(parsed);

    if (jackpotWins.length === 0) return [];

    for (const win of jackpotWins) {
      await sendNotification(
        {
          type: 'success',
          title: '로또 1등 당첨!',
          message: `${win.roundNumber}회차 로또 ${win.rank}등 당첨을 확인했습니다.`,
          details: {
            roundNumber: win.roundNumber,
            rank: win.rank,
            prizeAmount: win.prizeAmount,
            prizeAmountKrw: `${formatKoreanNumber(win.prizeAmount)}원`,
            matchCount: win.matchCount,
            period: `${startDate} ~ ${endDate}`,
          },
        },
        env
      );
    }

    return jackpotWins;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.error('Winning check failed (non-fatal):', message);
    return [];
  }
}
