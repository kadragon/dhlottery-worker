/**
 * Account Information Retrieval Tests
 *
 * Trace:
 *   spec_id: SPEC-ACCOUNT-001
 *   task_id: TASK-003
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { getAccountInfo } from "./account";
import { createHttpClient } from "../client/http";
import { DHLotteryError } from "../utils/errors";
import type { HttpResponse } from "../types/http.types";
import { readFileSync } from "node:fs";
import { join } from "node:path";

describe("Account Information Retrieval", () => {
  let mockHttpClient: ReturnType<typeof createHttpClient>;
  let fixtureHTML: string;

  beforeEach(() => {
    // Create mock HTTP client
    mockHttpClient = {
      cookies: {},
      fetch: vi.fn(),
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    } as unknown as ReturnType<typeof createHttpClient>;

    // Load HTML fixture
    fixtureHTML = readFileSync(
      join(__dirname, "../__fixtures__/account-page.html"),
      "utf-8",
    );
  });

  /**
   * TEST-ACCOUNT-001: Should fetch account info page
   *
   * Criteria:
   * - GET request to account page succeeds
   * - Response status is 200
   * - Response contains HTML content
   */
  describe("TEST-ACCOUNT-001: Fetch account info page", () => {
    it("should fetch account info page successfully", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers({ "content-type": "text/html" }),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await getAccountInfo(mockHttpClient);

      // Assert
      expect(mockHttpClient.fetch).toHaveBeenCalledTimes(1);
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      expect(callArgs[0]).toContain("dhlottery.co.kr");
    });

    it("should send GET request to correct endpoint", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await getAccountInfo(mockHttpClient);

      // Assert
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      expect(callArgs[1]?.method).toBeUndefined(); // GET is default
    });

    it("should handle HTTP errors", async () => {
      // Arrange
      const mockResponse = {
        status: 500,
        statusText: "Internal Server Error",
        headers: new Headers(),
        text: async () => "Error",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
    });
  });

  /**
   * TEST-ACCOUNT-002: Should parse deposit balance correctly
   *
   * Criteria:
   * - Balance regex matches format: '예치금: N,NNN원'
   * - Commas are removed before parsing
   * - Result is number in KRW
   * - Zero balance is handled correctly
   */
  describe("TEST-ACCOUNT-002: Parse deposit balance", () => {
    it("should parse balance with commas", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML, // Contains "45,000원"
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(45000);
      expect(typeof result.balance).toBe("number");
    });

    it("should parse balance without commas", async () => {
      // Arrange
      const htmlWithoutCommas = fixtureHTML.replace("45,000", "5000");
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithoutCommas,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(5000);
    });

    it("should handle zero balance", async () => {
      // Arrange
      const htmlWithZero = fixtureHTML.replace("45,000", "0");
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithZero,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(0);
    });

    it("should handle large balances with multiple commas", async () => {
      // Arrange
      const htmlWithLargeBalance = fixtureHTML.replace("45,000", "1,234,567");
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithLargeBalance,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(1234567);
    });

    it("should throw error if balance not found", async () => {
      // Arrange
      const htmlWithoutBalance = "<html><body>No balance here</body></html>";
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithoutBalance,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        /balance/i,
      );
    });
  });

  /**
   * TEST-ACCOUNT-003: Should parse current lottery round
   *
   * Criteria:
   * - Round number is extracted from HTML
   * - Result is positive integer
   * - Invalid round throws error
   */
  describe("TEST-ACCOUNT-003: Parse lottery round", () => {
    it("should parse round number correctly", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML, // Contains "제1145회"
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.currentRound).toBe(1145);
      expect(typeof result.currentRound).toBe("number");
      expect(Number.isInteger(result.currentRound)).toBe(true);
    });

    it("should handle different round numbers", async () => {
      // Arrange
      const htmlWithDifferentRound = fixtureHTML.replace("1145", "999");
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithDifferentRound,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.currentRound).toBe(999);
    });

    it("should throw error if round not found", async () => {
      // Arrange - HTML with balance but no round
      const htmlWithoutRound =
        "<html><body><dd><strong>10,000</strong>원</dd></body></html>";
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithoutRound,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(/round/i);
    });
  });

  /**
   * TEST-ACCOUNT-004: Should validate parsed data
   *
   * Criteria:
   * - Balance is non-negative number
   * - Round is positive integer
   * - Missing data throws descriptive error
   */
  describe("TEST-ACCOUNT-004: Validate parsed data", () => {
    it("should validate balance is non-negative", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBeGreaterThanOrEqual(0);
    });

    it("should validate round is positive", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.currentRound).toBeGreaterThan(0);
    });

    it("should throw error for negative balance", async () => {
      // Arrange - malformed HTML that would parse to negative
      const htmlWithInvalidBalance = fixtureHTML.replace(
        "<strong>45,000</strong>",
        "<strong>-1000</strong>",
      );
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithInvalidBalance,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
    });

    it("should throw error for zero or negative round", async () => {
      // Arrange
      const htmlWithInvalidRound = fixtureHTML.replace(
        "<strong>1145</strong>",
        "<strong>0</strong>",
      );
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => htmlWithInvalidRound,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
    });
  });

  /**
   * TEST-ACCOUNT-005: Should return structured account info
   *
   * Criteria:
   * - Returns object with balance and round
   * - Types match AccountInfo interface
   * - Data is immediately usable
   */
  describe("TEST-ACCOUNT-005: Return structured account info", () => {
    it("should return object with balance and currentRound", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result).toHaveProperty("balance");
      expect(result).toHaveProperty("currentRound");
      expect(Object.keys(result)).toHaveLength(2);
    });

    it("should return properly typed data", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(typeof result.balance).toBe("number");
      expect(typeof result.currentRound).toBe("number");
      expect(Number.isInteger(result.currentRound)).toBe(true);
    });

    it("should return immediately usable data", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => fixtureHTML,
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert - should be able to perform calculations
      const canAfford = result.balance >= 5000;
      expect(typeof canAfford).toBe("boolean");

      const nextRound = result.currentRound + 1;
      expect(nextRound).toBe(1146);
    });
  });
});
