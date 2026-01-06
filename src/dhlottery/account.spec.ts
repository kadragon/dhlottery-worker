/**
 * Account Information Retrieval Tests
 *
 * Trace:
 *   spec_id: SPEC-ACCOUNT-001, SPEC-ACCOUNT-003
 *   task_id: TASK-003, TASK-011, TASK-ROUND-API-UPDATE
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { getAccountInfo } from "./account";
import { createHttpClient } from "../client/http";
import { DHLotteryError } from "../utils/errors";
import type { HttpResponse } from "../types";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Load JSON fixture for new balance API
const balanceApiResponse = JSON.parse(
  readFileSync(
    join(__dirname, "../__fixtures__/selectUserMndp-response.json"),
    "utf-8",
  ),
);

describe("Account Information Retrieval", () => {
  let mockHttpClient: ReturnType<typeof createHttpClient>;

  // Default round API response
  const defaultRoundApiResponse = {
    resultCode: null,
    resultMessage: null,
    data: {
      result: {
        ltEpsd: 1206,
        ltRflYmd: "20260110",
        ltRflHh: "20",
        ltRflMm: "00",
      },
      gameMng: null,
    },
  };

  // Default balance API response (matches fixture)
  const defaultBalanceApiResponse = balanceApiResponse;

  beforeEach(() => {
    // Create mock HTTP client
    mockHttpClient = {
      cookies: {},
      fetch: vi.fn(),
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    } as unknown as ReturnType<typeof createHttpClient>;
  });

  /**
   * Helper to mock account API responses (both JSON)
   * 2026-01 Update: Uses JSON API for both round and balance
   */
  const mockAccountResponses = (
    roundData: typeof defaultRoundApiResponse,
    balanceData: typeof defaultBalanceApiResponse
  ) => {
    // Round API Response (JSON)
    const roundApiResponse = {
      status: 200,
      statusText: "OK",
      headers: new Headers(),
      text: async () => JSON.stringify(roundData),
      json: async () => roundData,
    } as unknown as HttpResponse;

    // Balance API Response (JSON)
    const balanceResponse = {
      status: 200,
      statusText: "OK",
      headers: new Headers(),
      text: async () => JSON.stringify(balanceData),
      json: async () => balanceData,
    } as unknown as HttpResponse;

    vi.mocked(mockHttpClient.fetch)
      .mockResolvedValueOnce(roundApiResponse)
      .mockResolvedValueOnce(balanceResponse);
  };

  /**
   * TEST-ACCOUNT-001: Should fetch account info
   *
   * Criteria:
   * - GET request to round API and balance API succeeds
   * - Response status is 200
   * - Round from JSON API, balance from JSON API
   *
   * 2026-01 Update: Uses JSON API for both round and balance
   */
  describe("TEST-ACCOUNT-001: Fetch account info", () => {
    it("should fetch round API then balance API successfully", async () => {
      // Arrange
      mockAccountResponses(defaultRoundApiResponse, defaultBalanceApiResponse);

      // Act
      await getAccountInfo(mockHttpClient);

      // Assert - Should call Round API then Balance API
      expect(mockHttpClient.fetch).toHaveBeenCalledTimes(2);

      const firstCall = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      expect(firstCall[0]).toContain("/lt645/selectThsLt645Info.do");

      const secondCall = vi.mocked(mockHttpClient.fetch).mock.calls[1];
      expect(secondCall[0]).toContain("/mypage/selectUserMndp.do");
    });

    it("should send GET request to correct endpoint", async () => {
      // Arrange
      mockAccountResponses(defaultRoundApiResponse, defaultBalanceApiResponse);

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

    it("should include stage and redirect location when round API returns 302", async () => {
      // Arrange
      const redirectLocation = "https://dhlottery.co.kr/";
      const roundApiResponse = {
        status: 302,
        statusText: "Found",
        headers: new Headers({ Location: redirectLocation }),
        text: async () => "",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(roundApiResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrowError(
        /account[\s\S]*round API[\s\S]*HTTP 302[\s\S]*Location:/i,
      );
    });

    it("should throw error if balance API returns 302", async () => {
      // Arrange
      const roundApiResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify(defaultRoundApiResponse),
        json: async () => defaultRoundApiResponse,
      } as unknown as HttpResponse;

      const balanceApiResponseError = {
        status: 302,
        statusText: "Found",
        headers: new Headers({ Location: "https://dhlottery.co.kr/login" }),
        text: async () => "",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(roundApiResponse)
        .mockResolvedValueOnce(balanceApiResponseError);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrowError(
        /account[\s\S]*balance API[\s\S]*HTTP 302/i,
      );
    });
  });

  /**
   * TEST-ACCOUNT-002: Should parse deposit balance correctly
   *
   * Criteria:
   * - Balance is parsed from JSON API crntEntrsAmt field
   * - Result is number in KRW
   * - Zero balance is handled correctly
   *
   * 2026-01 Update: Uses JSON API instead of HTML parsing
   */
  describe("TEST-ACCOUNT-002: Parse deposit balance", () => {
    it("should parse balance from JSON API", async () => {
      // Arrange - API returns crntEntrsAmt: 20000
      mockAccountResponses(defaultRoundApiResponse, defaultBalanceApiResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(20000);
      expect(typeof result.balance).toBe("number");
    });

    it("should parse different balance values", async () => {
      // Arrange
      const customBalanceResponse = {
        ...defaultBalanceApiResponse,
        data: {
          userMndp: { ...defaultBalanceApiResponse.data.userMndp, crntEntrsAmt: 5000 },
        },
      };
      mockAccountResponses(defaultRoundApiResponse, customBalanceResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(5000);
    });

    it("should handle zero balance", async () => {
      // Arrange
      const zeroBalanceResponse = {
        ...defaultBalanceApiResponse,
        data: {
          userMndp: { ...defaultBalanceApiResponse.data.userMndp, crntEntrsAmt: 0 },
        },
      };
      mockAccountResponses(defaultRoundApiResponse, zeroBalanceResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(0);
    });

    it("should handle large balances", async () => {
      // Arrange
      const largeBalanceResponse = {
        ...defaultBalanceApiResponse,
        data: {
          userMndp: { ...defaultBalanceApiResponse.data.userMndp, crntEntrsAmt: 1234567 },
        },
      };
      mockAccountResponses(defaultRoundApiResponse, largeBalanceResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(1234567);
    });

    it("should throw error if crntEntrsAmt not found", async () => {
      // Arrange - round API works, but balance API has no crntEntrsAmt
      const invalidBalanceResponse = {
        resultCode: null,
        resultMessage: null,
        data: { userMndp: {} },
      };
      mockAccountResponses(defaultRoundApiResponse, invalidBalanceResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
    });
  });

  /**
   * TEST-ACCOUNT-003: Should parse current lottery round
   *
   * Criteria:
   * - Round number is extracted from JSON API (ltEpsd field)
   * - Result is positive integer
   * - Invalid round throws error
   *
   * 2026-01 Update: Uses /lt645/selectThsLt645Info.do JSON API
   */
  describe("TEST-ACCOUNT-003: Parse lottery round", () => {
    it("should parse round number from API correctly", async () => {
      // Arrange - API returns round 1206
      mockAccountResponses(defaultRoundApiResponse, defaultBalanceApiResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert - ltEpsd is 1206
      expect(result.currentRound).toBe(1206);
      expect(typeof result.currentRound).toBe("number");
      expect(Number.isInteger(result.currentRound)).toBe(true);
    });

    it("should handle different round numbers", async () => {
      // Arrange - API with different round number
      const customRoundResponse = {
        ...defaultRoundApiResponse,
        data: {
          ...defaultRoundApiResponse.data,
          result: { ...defaultRoundApiResponse.data.result, ltEpsd: 998 },
        },
      };
      mockAccountResponses(customRoundResponse, defaultBalanceApiResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.currentRound).toBe(998);
    });

    it("should throw error if round not found in API response", async () => {
      // Arrange - API without round number
      const invalidRoundResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => "{}",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockReset();
      vi.mocked(mockHttpClient.fetch).mockResolvedValue(invalidRoundResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(DHLotteryError);

      // Reset again for second assertion
      vi.mocked(mockHttpClient.fetch).mockReset();
      vi.mocked(mockHttpClient.fetch).mockResolvedValue(invalidRoundResponse);
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(/round/i);
    });
  });

  /**
   * TEST-ACCOUNT-004: Should validate parsed data
   *
   * Criteria:
   * - Balance must be non-negative
   * - Round must be positive
   * - Invalid data throws error
   */
  describe("TEST-ACCOUNT-004: Validate parsed data", () => {
    it("should validate balance is non-negative", async () => {
      // Arrange
      mockAccountResponses(defaultRoundApiResponse, defaultBalanceApiResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBeGreaterThanOrEqual(0);
    });

    it("should validate round is positive", async () => {
      // Arrange
      mockAccountResponses(defaultRoundApiResponse, defaultBalanceApiResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.currentRound).toBeGreaterThan(0);
    });

    it("should throw error for negative balance from API", async () => {
      // Arrange - API returns negative balance
      const negativeBalanceResponse = {
        ...defaultBalanceApiResponse,
        data: {
          userMndp: { ...defaultBalanceApiResponse.data.userMndp, crntEntrsAmt: -1000 },
        },
      };
      mockAccountResponses(defaultRoundApiResponse, negativeBalanceResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
    });

    it("should throw error for zero or negative round from API", async () => {
      // Arrange - API returns zero round
      const zeroRoundResponse = {
        ...defaultRoundApiResponse,
        data: {
          ...defaultRoundApiResponse.data,
          result: { ...defaultRoundApiResponse.data.result, ltEpsd: 0 },
        },
      };

      const roundApiResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify(zeroRoundResponse),
        json: async () => zeroRoundResponse,
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockReset();
      vi.mocked(mockHttpClient.fetch).mockResolvedValue(roundApiResponse);

      // Act & Assert
      await expect(getAccountInfo(mockHttpClient)).rejects.toThrow(
        DHLotteryError,
      );
    });

    /**
     * TEST-ACCOUNT-005: Integration test with both APIs
     */
    it("should parse both balance and round correctly", async () => {
      // Arrange
      mockAccountResponses(defaultRoundApiResponse, defaultBalanceApiResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert - should be able to perform calculations
      const canAfford = result.balance >= 5000;
      expect(typeof canAfford).toBe("boolean");

      // API returns 1206 as the current round
      expect(result.currentRound).toBe(1206);
      expect(result.balance).toBe(20000);
    });
  });

  /**
   * TEST-ACCOUNT-008: Should parse balance from selectUserMndp.do JSON API
   *
   * Criteria:
   * - Uses /mypage/selectUserMndp.do API endpoint
   * - Parses crntEntrsAmt field from JSON response
   * - Returns balance as number
   *
   * 2026-01 Update: mypage/home is JS-rendered, use JSON API instead
   */
  describe("TEST-ACCOUNT-008: Parse balance from JSON API", () => {
    /**
     * Helper to mock both round API and balance API responses (both JSON)
     */
    const mockJsonApiResponses = (
      roundData: typeof defaultRoundApiResponse,
      balanceData: typeof balanceApiResponse
    ) => {
      // Round API Response
      const roundApiResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify(roundData),
        json: async () => roundData,
      } as unknown as HttpResponse;

      // Balance API Response
      const balanceResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify(balanceData),
        json: async () => balanceData,
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(roundApiResponse)
        .mockResolvedValueOnce(balanceResponse);
    };

    it("should fetch balance from selectUserMndp.do API", async () => {
      // Arrange
      mockJsonApiResponses(defaultRoundApiResponse, balanceApiResponse);

      // Act
      await getAccountInfo(mockHttpClient);

      // Assert - Should call Round API then Balance API
      expect(mockHttpClient.fetch).toHaveBeenCalledTimes(2);

      const secondCall = vi.mocked(mockHttpClient.fetch).mock.calls[1];
      expect(secondCall[0]).toContain("/mypage/selectUserMndp.do");
    });

    it("should parse crntEntrsAmt from JSON response", async () => {
      // Arrange
      mockJsonApiResponses(defaultRoundApiResponse, balanceApiResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(20000);
      expect(typeof result.balance).toBe("number");
    });

    it("should handle zero balance from API", async () => {
      // Arrange
      const zeroBalanceResponse = {
        ...balanceApiResponse,
        data: {
          userMndp: { ...balanceApiResponse.data.userMndp, crntEntrsAmt: 0 },
        },
      };
      mockJsonApiResponses(defaultRoundApiResponse, zeroBalanceResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(0);
    });

    it("should handle large balance from API", async () => {
      // Arrange
      const largeBalanceResponse = {
        ...balanceApiResponse,
        data: {
          userMndp: { ...balanceApiResponse.data.userMndp, crntEntrsAmt: 5000000 },
        },
      };
      mockJsonApiResponses(defaultRoundApiResponse, largeBalanceResponse);

      // Act
      const result = await getAccountInfo(mockHttpClient);

      // Assert
      expect(result.balance).toBe(5000000);
    });
  });
});
