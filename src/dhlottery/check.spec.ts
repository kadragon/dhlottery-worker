/**
 * Winning Check Tests
 *
 * Trace:
 *   spec_id: SPEC-WINNING-001
 *   task_id: TASK-006, TASK-011
 */

import { readFileSync } from "node:fs";
import { join } from "node:path";
import { beforeEach, describe, expect, it, vi, afterEach } from "vitest";
import type { HttpClient, HttpResponse } from "../types";
import {
  WINNING_PATTERNS,
  calculatePreviousWeekRange,
  checkWinning,
  filterJackpotWins,
  parseWinningResultsFromHtml,
} from "./check";

vi.mock("../notify/telegram", () => ({
  sendNotification: vi.fn(),
}));

describe("Winning Check", () => {
  let fixtureHTML: string;
  let mockHttpClient: HttpClient;

  beforeEach(() => {
    // Mock process.env
    vi.stubEnv('TELEGRAM_BOT_TOKEN', '123:token');
    vi.stubEnv('TELEGRAM_CHAT_ID', '987654321');

    fixtureHTML = readFileSync(
      join(__dirname, "../__fixtures__/winning-results.html"),
      "utf-8",
    );

    mockHttpClient = {
      cookies: {},
      fetch: vi.fn(),
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    };

    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  /**
   * TEST-WINNING-001: Should calculate correct date range
   *
   * Criteria:
   * - Start date = previous Monday
   * - End date = previous Sunday
   * - Period covers exactly 7 days
   */
  describe("TEST-WINNING-001: Calculate previous week range", () => {
    it("should calculate previous Monday to Sunday in KST", () => {
      const nowKstMonday = new Date("2025-12-15T10:00:00+09:00"); // Monday (KST)
      const range = calculatePreviousWeekRange(nowKstMonday);

      expect(range.startDate).toBe("2025-12-08");
      expect(range.endDate).toBe("2025-12-14");

      const oneDay = 24 * 60 * 60 * 1000;
      expect(range.end.getTime() - range.start.getTime() + 1).toBe(7 * oneDay);
    });

    it("should throw for invalid now date", () => {
      expect(() => calculatePreviousWeekRange(new Date("invalid"))).toThrow();
    });
  });

  /**
   * TEST-WINNING-002: Should fetch winning results page
   *
   * Criteria:
   * - GET request to winning check endpoint
   * - Date range parameters included
   * - Response contains result table HTML
   */
  describe("TEST-WINNING-002: Fetch winning results page", () => {
    it("should call GET endpoint with date range params", async () => {
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers({ "content-type": "text/html" }),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      await checkWinning(mockHttpClient, new Date("2025-12-15T10:00:00+09:00"));

      expect(mockHttpClient.fetch).toHaveBeenCalledTimes(1);
      const [url, options] = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      expect(url).toContain("myPage.do");
      expect(url).toContain("method=lottoBuyList");
      expect(url).toContain("searchStartDate=");
      expect(url).toContain("searchEndDate=");
      expect(options?.method).toBeUndefined(); // default GET
    });
  });

  /**
   * TEST-WINNING-003: Should parse winning results correctly
   *
   * Criteria:
   * - Extract lottery round numbers
   * - Extract winning ranks (1, 2, 3, etc.)
   * - Extract prize amounts
   * - Parse each table row independently
   */
  describe("TEST-WINNING-003: Parse winning results", () => {
    it("should parse round, rank, prize, matchCount", () => {
      const results = parseWinningResultsFromHtml(fixtureHTML);

      expect(results.length).toBe(2);
      expect(results[0]).toMatchObject({
        roundNumber: 1144,
        rank: 1,
        prizeAmount: 2000000000,
        matchCount: 6,
      });
      expect(results[1]).toMatchObject({
        roundNumber: 1144,
        rank: 2,
        prizeAmount: 65000000,
        matchCount: 5,
      });
    });
  });

  /**
   * TEST-WINNING-004: Should filter for rank 1 and higher only
   *
   * Criteria:
   * - Rank 1 wins included
   * - Rank > 1 excluded from notifications
   * - Empty results handled gracefully
   */
  describe("TEST-WINNING-004: Filter rank 1 only", () => {
    it("should keep only rank 1 results", () => {
      const results = parseWinningResultsFromHtml(fixtureHTML);
      const filtered = filterJackpotWins(results);

      expect(filtered.length).toBe(1);
      expect(filtered[0]?.rank).toBe(1);
    });

    it("should handle empty list", () => {
      expect(filterJackpotWins([])).toEqual([]);
    });
  });

  /**
   * TEST-WINNING-005: Should notify wins via Telegram
   *
   * Criteria:
   * - Notification type = 'success'
   * - Message includes round number
   * - Message includes rank
   * - Message includes prize amount
   */
  describe("TEST-WINNING-005: Notify wins", () => {
    it("should call sendNotification for rank 1 win", async () => {
      const { sendNotification } = await import("../notify/telegram");

      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers({ "content-type": "text/html" }),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      const results = await checkWinning(
        mockHttpClient,
        new Date("2025-12-15T10:00:00+09:00"),
      );

      expect(results.length).toBe(1);
      expect(vi.mocked(sendNotification)).toHaveBeenCalledTimes(1);

      const [payload] = vi.mocked(sendNotification).mock.calls[0];
      expect(payload.type).toBe("success");
      expect(JSON.stringify(payload)).toContain("1144");
      expect(JSON.stringify(payload)).toContain("1");
      expect(JSON.stringify(payload)).toContain("2000000000");
    });
  });

  /**
   * TEST-WINNING-007: Global regex lastIndex bug - multiple rows
   *
   * Criteria:
   * - Parse multiple table rows correctly
   * - Each row must have all cells extracted
   * - Shared global regex must not interfere between rows
   */
  describe("TEST-WINNING-007: Global regex should not break multi-row parsing", () => {
    it("should demonstrate why matchAll is needed instead of exec", () => {
      // This test documents the global regex lastIndex bug that would occur
      // if we used .exec() in a loop with a shared global regex.
      // Our fix (using .matchAll()) avoids this issue entirely.

      const regex = WINNING_PATTERNS.tableCell;
      const row1 = '<td>A</td><td>B</td><td>C</td>';
      const row2 = '<td>D</td><td>E</td><td>F</td>';

      // Using .exec() with a global regex causes lastIndex to persist
      const cells1: string[] = [];
      let match1 = regex.exec(row1);
      if (match1) cells1.push(match1[1] ?? '');
      match1 = regex.exec(row1);
      if (match1) cells1.push(match1[1] ?? '');
      match1 = regex.exec(row1);
      if (match1) cells1.push(match1[1] ?? '');

      // lastIndex is now at the end of row1
      const lastIndexAfterRow1 = regex.lastIndex;
      expect(lastIndexAfterRow1).toBeGreaterThan(0);

      // Using .exec() on row2 without reset causes partial matches
      const cells2WithBug: string[] = [];
      let match2 = regex.exec(row2);
      if (match2) cells2WithBug.push(match2[1] ?? '');
      match2 = regex.exec(row2);
      if (match2) cells2WithBug.push(match2[1] ?? '');
      match2 = regex.exec(row2);
      if (match2) cells2WithBug.push(match2[1] ?? '');

      // Demonstrates the bug: only partial matches due to stale lastIndex
      expect(cells1).toEqual(['A', 'B', 'C']);
      expect(cells2WithBug.length).toBeLessThan(3); // Bug reproduced

      // The fix: using .matchAll() always works correctly
      regex.lastIndex = 0; // Reset for clean test
      const cells2Fixed = Array.from(row2.matchAll(regex), (m) => m[1] ?? '');
      expect(cells2Fixed).toEqual(['D', 'E', 'F']); // Fix verified
    });

    it("should parse all cells from multiple rows without lastIndex interference", () => {
      const multiRowHtml = `
        <tr>
          <td>2025-12-10</td>
          <td>로또6/45</td>
          <td><a href="javascript:detailPop('x','y','1140');">상세</a></td>
          <td>5</td>
          <td>1등 (일치 6개)</td>
          <td>1,000,000,000원</td>
          <td>2025-12-13</td>
        </tr>
        <tr>
          <td>2025-12-11</td>
          <td>로또6/45</td>
          <td><a href="javascript:detailPop('x','y','1141');">상세</a></td>
          <td>3</td>
          <td>2등 (일치 5개)</td>
          <td>50,000,000원</td>
          <td>2025-12-14</td>
        </tr>
        <tr>
          <td>2025-12-12</td>
          <td>로또6/45</td>
          <td><a href="javascript:detailPop('x','y','1142');">상세</a></td>
          <td>2</td>
          <td>3등 (일치 5개)</td>
          <td>1,500,000원</td>
          <td>2025-12-15</td>
        </tr>
      `;

      const results = parseWinningResultsFromHtml(multiRowHtml);

      // All 3 rows should be parsed, not just the first one
      expect(results.length).toBe(3);
      expect(results[0]?.roundNumber).toBe(1140);
      expect(results[1]?.roundNumber).toBe(1141);
      expect(results[2]?.roundNumber).toBe(1142);
    });
  });

  /**
   * TEST-WINNING-006: Should handle no-win scenario gracefully
   *
   * Criteria:
   * - Empty results array returned
   * - No Telegram notification sent
   * - Execution continues normally
   */
  describe("TEST-WINNING-006: No-win scenario", () => {
    it("should return empty and not notify when no rank 1 wins", async () => {
      const { sendNotification } = await import("../notify/telegram");

      const noWinHtml = fixtureHTML.replace("1등", "2등").replace("2등", "3등");

      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers({ "content-type": "text/html" }),
        text: async () => noWinHtml,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      const results = await checkWinning(
        mockHttpClient,
        new Date("2025-12-15T10:00:00+09:00"),
      );

      expect(results).toEqual([]);
      expect(vi.mocked(sendNotification)).not.toHaveBeenCalled();
    });

    it("should handle fetch failure without throwing", async () => {
      vi.mocked(mockHttpClient.fetch).mockResolvedValue({
        status: 500,
        statusText: "Internal Server Error",
        headers: new Headers(),
        text: async () => "error",
        json: async () => ({}),
      } as unknown as HttpResponse);

      await expect(
        checkWinning(mockHttpClient, new Date("2025-12-15T10:00:00+09:00")),
      ).resolves.toEqual([]);
    });

    it("should handle parsing failure without throwing", async () => {
      vi.mocked(mockHttpClient.fetch).mockResolvedValue({
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => "<html>no rows</html>",
        json: async () => ({}),
      } as unknown as HttpResponse);

      await expect(
        checkWinning(mockHttpClient, new Date("2025-12-15T10:00:00+09:00")),
      ).resolves.toEqual([]);
    });
  });
});
