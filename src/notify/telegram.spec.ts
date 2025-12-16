/**
 * Telegram Notification Service Tests
 *
 * Trace:
 *   spec_id: SPEC-TELEGRAM-001
 *   task_id: TASK-007
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { sendNotification } from "./telegram";
import type { TelegramEnv } from "../types/notification.types";

describe("Telegram Notification Service", () => {
  let mockFetch: ReturnType<typeof vi.fn>;
  let mockEnv: TelegramEnv;

  beforeEach(() => {
    // Mock global fetch
    mockFetch = vi.fn();
    global.fetch = mockFetch;

    // Create mock environment with Telegram credentials
    mockEnv = {
      TELEGRAM_BOT_TOKEN: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
      TELEGRAM_CHAT_ID: "987654321",
    };
  });

  /**
   * TEST-TELEGRAM-001: Should send message with correct API endpoint
   *
   * Criteria:
   * - POST to https://api.telegram.org/bot{token}/sendMessage
   * - Request includes chat_id and text
   * - parse_mode set to 'Markdown' or 'MarkdownV2'
   */
  describe("TEST-TELEGRAM-001: Send message to correct API endpoint", () => {
    it("should send POST request to correct Telegram API URL", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true, result: {} }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Test",
          message: "Test message",
        },
        mockEnv,
      );

      // Assert
      expect(mockFetch).toHaveBeenCalledTimes(1);
      const url = mockFetch.mock.calls[0][0];
      expect(url).toContain("https://api.telegram.org/bot");
      expect(url).toContain(mockEnv.TELEGRAM_BOT_TOKEN);
      expect(url).toContain("/sendMessage");
    });

    it("should include chat_id and text in request body", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Test Title",
          message: "Test Message",
        },
        mockEnv,
      );

      // Assert
      const requestOptions = mockFetch.mock.calls[0][1];
      expect(requestOptions.method).toBe("POST");

      const requestBody = JSON.parse(requestOptions.body);
      expect(requestBody).toHaveProperty("chat_id");
      expect(requestBody.chat_id).toBe(mockEnv.TELEGRAM_CHAT_ID);
      expect(requestBody).toHaveProperty("text");
      expect(requestBody.text).toContain("Test Title");
    });

    it("should set parse_mode to Markdown", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Test",
          message: "Test",
        },
        mockEnv,
      );

      // Assert
      const requestOptions = mockFetch.mock.calls[0][1];
      const requestBody = JSON.parse(requestOptions.body);
      expect(requestBody.parse_mode).toMatch(/Markdown/);
    });
  });

  /**
   * TEST-TELEGRAM-002: Should format purchase success notification
   *
   * Criteria:
   * - Message type = 'success'
   * - Title indicates purchase completion
   * - Body includes: game count, cost, round number
   * - Formatting is clear and readable
   */
  describe("TEST-TELEGRAM-002: Format purchase success notification", () => {
    it("should format purchase success with all details", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Lottery Purchase Completed",
          message: "Successfully purchased lottery tickets",
          details: {
            gameCount: 5,
            totalCost: 5000,
            roundNumber: 1145,
          },
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Lottery Purchase Completed");
      expect(text).toContain("5"); // game count
      expect(text).toContain("5000"); // cost
      expect(text).toContain("1145"); // round
    });

    it("should include success emoji or indicator", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Purchase Complete",
          message: "Done",
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/✅|✓|success/i);
    });
  });

  /**
   * TEST-TELEGRAM-003: Should format low balance warning
   *
   * Criteria:
   * - Message type = 'warning'
   * - Title indicates insufficient balance
   * - Body includes: current balance, minimum required
   * - Instructions for manual deposit
   */
  describe("TEST-TELEGRAM-003: Format low balance warning", () => {
    it("should format low balance warning with all details", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "warning",
          title: "Insufficient Balance",
          message: "Please deposit funds manually",
          details: {
            currentBalance: 2000,
            minimumBalance: 30000,
          },
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Insufficient Balance");
      expect(text).toContain("2000"); // current
      expect(text).toContain("30000"); // minimum
      expect(text).toMatch(/deposit|manually/i);
    });

    it("should include warning emoji or indicator", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "warning",
          title: "Warning",
          message: "Low balance",
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/⚠️|⚠|warning/i);
    });
  });

  /**
   * TEST-TELEGRAM-004: Should format error notification
   *
   * Criteria:
   * - Message type = 'error'
   * - Title indicates error occurred
   * - Body includes: error type, error message
   * - Optional: stack trace or context (sanitized)
   */
  describe("TEST-TELEGRAM-004: Format error notification", () => {
    it("should format error notification with details", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "error",
          title: "Error Occurred",
          message: "Authentication failed",
          details: {
            errorType: "AuthenticationError",
            errorCode: "AUTH_INVALID_CREDENTIALS",
          },
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Error Occurred");
      expect(text).toContain("Authentication failed");
      expect(text).toContain("AuthenticationError");
    });

    it("should include error emoji or indicator", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "error",
          title: "Error",
          message: "Something went wrong",
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/❌|✗|error/i);
    });
  });

  /**
   * TEST-TELEGRAM-005: Should format winning notification
   *
   * Criteria:
   * - Message type = 'success'
   * - Title indicates winning detected
   * - Body includes: round number, rank, prize amount
   */
  describe("TEST-TELEGRAM-005: Format winning notification", () => {
    it("should format winning notification with all details", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Winning Detected!",
          message: "Congratulations on your win",
          details: {
            roundNumber: 1144,
            rank: 1,
            prizeAmount: 2000000000,
          },
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Winning Detected");
      expect(text).toContain("1144"); // round
      expect(text).toContain("1"); // rank
      expect(text).toContain("2000000000"); // prize
    });

    it("should include celebration emoji", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "You Won!",
          message: "Winner!",
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/🎉|🎊|🏆|win/i);
    });
  });

  /**
   * TEST-TELEGRAM-006: Should use credentials from Secrets
   *
   * Criteria:
   * - TELEGRAM_BOT_TOKEN from env.TELEGRAM_BOT_TOKEN
   * - TELEGRAM_CHAT_ID from env.TELEGRAM_CHAT_ID
   * - No hardcoded tokens in code
   */
  describe("TEST-TELEGRAM-006: Use credentials from Secrets", () => {
    it("should use bot token from environment", async () => {
      // Arrange
      const customToken = "999888:CustomTokenForTesting";
      mockEnv.TELEGRAM_BOT_TOKEN = customToken;

      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Test",
          message: "Test",
        },
        mockEnv,
      );

      // Assert
      const url = mockFetch.mock.calls[0][0];
      expect(url).toContain(customToken);
    });

    it("should use chat ID from environment", async () => {
      // Arrange
      const customChatId = "123123123";
      mockEnv.TELEGRAM_CHAT_ID = customChatId;

      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Test",
          message: "Test",
        },
        mockEnv,
      );

      // Assert
      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.chat_id).toBe(customChatId);
    });
  });

  /**
   * TEST-TELEGRAM-007: Should handle API failures gracefully
   *
   * Criteria:
   * - Network errors are caught
   * - API errors (4xx, 5xx) are logged
   * - Error does not crash main execution
   */
  describe("TEST-TELEGRAM-007: Handle API failures gracefully", () => {
    it("should handle network errors without throwing", async () => {
      // Arrange
      mockFetch.mockRejectedValue(new Error("Network error"));

      // Act & Assert - should not throw
      await expect(
        sendNotification(
          {
            type: "success",
            title: "Test",
            message: "Test",
          },
          mockEnv,
        ),
      ).resolves.not.toThrow();
    });

    it("should handle 4xx errors without throwing", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        statusText: "Bad Request",
        json: async () => ({ ok: false, error_code: 400, description: "Bad Request" }),
      });

      // Act & Assert - should not throw
      await expect(
        sendNotification(
          {
            type: "success",
            title: "Test",
            message: "Test",
          },
          mockEnv,
        ),
      ).resolves.not.toThrow();
    });

    it("should handle 5xx errors without throwing", async () => {
      // Arrange
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: async () => ({ ok: false }),
      });

      // Act & Assert - should not throw
      await expect(
        sendNotification(
          {
            type: "error",
            title: "Test",
            message: "Test",
          },
          mockEnv,
        ),
      ).resolves.not.toThrow();
    });

    it("should log errors using console.error", async () => {
      // Arrange
      const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      mockFetch.mockRejectedValue(new Error("Network failure"));

      // Act
      await sendNotification(
        {
          type: "success",
          title: "Test",
          message: "Test",
        },
        mockEnv,
      );

      // Assert
      expect(consoleErrorSpy).toHaveBeenCalled();

      // Cleanup
      consoleErrorSpy.mockRestore();
    });
  });
});
