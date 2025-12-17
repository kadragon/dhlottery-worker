/**
 * Winning Check Tests
 *
 * Trace:
 *   spec_id: SPEC-WINNING-001
 *   task_id: TASK-006, TASK-011
 */

import { readFileSync } from "node:fs";
import { join } from "node:path";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { HttpClient, HttpResponse, TelegramEnv } from "../types";
import {
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
  let mockEnv: TelegramEnv;

  beforeEach(() => {
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

    mockEnv = {
      TELEGRAM_BOT_TOKEN: "123:token",
      TELEGRAM_CHAT_ID: "987654321",
    };

    vi.clearAllMocks();
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

      await checkWinning(mockHttpClient, mockEnv, new Date("2025-12-15T10:00:00+09:00"));

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
        mockEnv,
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
        mockEnv,
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
        checkWinning(mockHttpClient, mockEnv, new Date("2025-12-15T10:00:00+09:00")),
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
        checkWinning(mockHttpClient, mockEnv, new Date("2025-12-15T10:00:00+09:00")),
      ).resolves.toEqual([]);
    });
  });
});
