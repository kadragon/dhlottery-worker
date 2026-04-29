/**
 * Telegram Notification Service Tests
 *
 * Trace:
 *   spec_id: SPEC-TELEGRAM-001
 *   task_id: TASK-007, TASK-011
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { sendNotification, sendCombinedNotification } from "./telegram";

describe("Telegram Notification Service", () => {
  let mockFetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // Mock process.env
    vi.stubEnv('TELEGRAM_BOT_TOKEN', '123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11');
    vi.stubEnv('TELEGRAM_CHAT_ID', '987654321');

    // Mock global fetch
    mockFetch = vi.fn();
    global.fetch = mockFetch as any;
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.useRealTimers();
  });

  /**
   * TEST-TELEGRAM-001: Should send message with correct API endpoint
   */
  describe("TEST-TELEGRAM-001: Send message to correct API endpoint", () => {
    it("should send POST request to correct Telegram API URL", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true, result: {} }),
      });

      await sendNotification({ type: "success", title: "Test", message: "Test message" });

      expect(mockFetch).toHaveBeenCalledTimes(1);
      const url = mockFetch.mock.calls[0][0];
      expect(url).toContain("https://api.telegram.org/bot");
      expect(url).toContain("123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11");
      expect(url).toContain("/sendMessage");
    });

    it("should include chat_id and text in request body", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "success", title: "Test Title", message: "Test Message" });

      const requestOptions = mockFetch.mock.calls[0][1];
      expect(requestOptions.method).toBe("POST");

      const requestBody = JSON.parse(requestOptions.body);
      expect(requestBody).toHaveProperty("chat_id");
      expect(requestBody.chat_id).toBe("987654321");
      expect(requestBody).toHaveProperty("text");
      expect(requestBody.text).toContain("Test Title");
    });

    it("should set parse_mode to Markdown", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "success", title: "Test", message: "Test" });

      const requestOptions = mockFetch.mock.calls[0][1];
      const requestBody = JSON.parse(requestOptions.body);
      expect(requestBody.parse_mode).toMatch(/Markdown/);
    });
  });

  /**
   * TEST-TELEGRAM-002: Should format purchase success notification
   */
  describe("TEST-TELEGRAM-002: Format purchase success notification", () => {
    it("should format purchase success with all details", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({
        type: "success",
        title: "Lottery Purchase Completed",
        message: "Successfully purchased lottery tickets",
        details: { gameCount: 5, totalCost: 5000, roundNumber: 1145 },
      });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Lottery Purchase Completed");
      expect(text).toContain("5");
      expect(text).toContain("5000");
      expect(text).toContain("1145");
    });

    it("should include success emoji or indicator", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "success", title: "Purchase Complete", message: "Done" });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/✅|✓|success/i);
    });
  });

  /**
   * TEST-TELEGRAM-003: Should format low balance warning
   */
  describe("TEST-TELEGRAM-003: Format low balance warning", () => {
    it("should format low balance warning with all details", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({
        type: "warning",
        title: "Insufficient Balance",
        message: "Please deposit funds manually",
        details: { currentBalance: 2000, minimumBalance: 30000 },
      });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Insufficient Balance");
      expect(text).toContain("2000");
      expect(text).toContain("30000");
      expect(text).toMatch(/deposit|manually/i);
    });

    it("should include warning emoji or indicator", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "warning", title: "Warning", message: "Low balance" });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/⚠️|⚠|warning/i);
    });
  });

  /**
   * TEST-TELEGRAM-004: Should format error notification
   */
  describe("TEST-TELEGRAM-004: Format error notification", () => {
    it("should format error notification with details", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({
        type: "error",
        title: "Error Occurred",
        message: "Authentication failed",
        details: { errorType: "AuthenticationError", errorCode: "AUTH_INVALID_CREDENTIALS" },
      });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Error Occurred");
      expect(text).toContain("Authentication failed");
      expect(text).toContain("AuthenticationError");
    });

    it("should include error emoji or indicator", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "error", title: "Error", message: "Something went wrong" });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/❌|✗|error/i);
    });
  });

  /**
   * TEST-TELEGRAM-005: Should format winning notification
   */
  describe("TEST-TELEGRAM-005: Format winning notification", () => {
    it("should format winning notification with all details", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({
        type: "success",
        title: "Winning Detected!",
        message: "Congratulations on your win",
        details: { roundNumber: 1144, rank: 1, prizeAmount: 2000000000 },
      });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      const text = requestBody.text;

      expect(text).toContain("Winning Detected");
      expect(text).toContain("1144");
      expect(text).toContain("1");
      expect(text).toContain("2000000000");
    });

    it("should include celebration emoji", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "success", title: "You Won!", message: "Winner!" });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.text).toMatch(/🎉|🎊|🏆|win/i);
    });
  });

  /**
   * TEST-TELEGRAM-006: Should use credentials from Secrets
   */
  describe("TEST-TELEGRAM-006: Use credentials from Secrets", () => {
    it("should use bot token from environment", async () => {
      const customToken = "999888:CustomTokenForTesting";
      vi.stubEnv('TELEGRAM_BOT_TOKEN', customToken);

      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "success", title: "Test", message: "Test" });

      const url = mockFetch.mock.calls[0][0];
      expect(url).toContain(customToken);
    });

    it("should use chat ID from environment", async () => {
      const customChatId = "123123123";
      vi.stubEnv('TELEGRAM_CHAT_ID', customChatId);

      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      await sendNotification({ type: "success", title: "Test", message: "Test" });

      const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(requestBody.chat_id).toBe(customChatId);
    });
  });

  /**
   * TEST-TELEGRAM-007: Should handle API failures gracefully
   */
  describe("TEST-TELEGRAM-007: Handle API failures gracefully", () => {
    it("should return false and not throw on permanent 4xx error", async () => {
      // 400 is not in RETRY_STATUSES — fails immediately without retrying
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        statusText: "Bad Request",
        json: async () => ({ ok: false, error_code: 400, description: "Bad Request" }),
      });

      await expect(
        sendNotification({ type: "success", title: "Test", message: "Test" })
      ).resolves.toBe(false);

      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it("should retry on network errors and return false after all attempts", async () => {
      vi.useFakeTimers();
      mockFetch.mockRejectedValue(new Error("Network error"));

      const resultPromise = sendNotification({ type: "success", title: "Test", message: "Test" });
      await vi.runAllTimersAsync();
      const result = await resultPromise;

      expect(result).toBe(false);
      expect(mockFetch).toHaveBeenCalledTimes(3); // 1 initial + 2 retries
    });

    it("should retry on 5xx errors and return false after all attempts", async () => {
      vi.useFakeTimers();
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: async () => ({ ok: false }),
      });

      const resultPromise = sendNotification({ type: "error", title: "Test", message: "Test" });
      await vi.runAllTimersAsync();
      const result = await resultPromise;

      expect(result).toBe(false);
      expect(mockFetch).toHaveBeenCalledTimes(3);
    });

    it("should log errors using console.error", async () => {
      vi.useFakeTimers();
      const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      mockFetch.mockRejectedValue(new Error("Network failure"));

      const resultPromise = sendNotification({ type: "success", title: "Test", message: "Test" });
      await vi.runAllTimersAsync();
      await resultPromise;

      expect(consoleErrorSpy).toHaveBeenCalled();
      consoleErrorSpy.mockRestore();
    });
  });

  /**
   * TEST-TELEGRAM-008: Retry behavior
   */
  describe("TEST-TELEGRAM-008: Retry behavior", () => {
    it("should succeed on second attempt after transient error", async () => {
      vi.useFakeTimers();
      mockFetch
        .mockResolvedValueOnce({ ok: false, status: 503, json: async () => ({}) })
        .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ ok: true }) });

      const resultPromise = sendNotification({ type: "success", title: "Test", message: "Test" });
      await vi.runAllTimersAsync();
      const result = await resultPromise;

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it("should return true immediately on success without retrying", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ok: true }),
      });

      const result = await sendNotification({ type: "success", title: "Test", message: "Test" });

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it("should not retry on 401 (permanent auth error)", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 401,
        statusText: "Unauthorized",
        json: async () => ({ ok: false, error_code: 401, description: "Unauthorized" }),
      });

      const result = await sendNotification({ type: "success", title: "Test", message: "Test" });

      expect(result).toBe(false);
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it("should retry on 429 (rate limit)", async () => {
      vi.useFakeTimers();
      mockFetch
        .mockResolvedValueOnce({ ok: false, status: 429, json: async () => ({}) })
        .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ ok: true }) });

      const resultPromise = sendNotification({ type: "success", title: "Test", message: "Test" });
      await vi.runAllTimersAsync();
      const result = await resultPromise;

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });
});

describe("sendCombinedNotification", () => {
  let mockFetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.stubEnv('TELEGRAM_BOT_TOKEN', '123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11');
    vi.stubEnv('TELEGRAM_CHAT_ID', '987654321');

    mockFetch = vi.fn();
    global.fetch = mockFetch as any;
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.useRealTimers();
  });

  it("should format multiple payloads separated by ---", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    });

    await sendCombinedNotification([
      { type: 'success', title: 'First', message: 'First message' },
      { type: 'warning', title: 'Second', message: 'Second message' },
    ]);

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const requestBody = JSON.parse(mockFetch.mock.calls[0][1].body);
    const text = requestBody.text;

    expect(text).toContain('First');
    expect(text).toContain('Second');
    expect(text).toContain('---');
  });

  it("should send exactly one Telegram API request for multiple payloads", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    });

    await sendCombinedNotification([
      { type: 'success', title: 'A', message: 'msg A' },
      { type: 'error', title: 'B', message: 'msg B' },
      { type: 'warning', title: 'C', message: 'msg C' },
    ]);

    expect(mockFetch).toHaveBeenCalledTimes(1);
  });

  it("should return true and not call fetch when payload list is empty", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    });

    const result = await sendCombinedNotification([]);

    expect(result).toBe(true);
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("should return true on successful send", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true }),
    });

    const result = await sendCombinedNotification([
      { type: 'success', title: 'Test', message: 'msg' },
    ]);

    expect(result).toBe(true);
  });

  it("should return false after all retries fail", async () => {
    vi.useFakeTimers();
    mockFetch.mockRejectedValue(new Error("Network error"));

    const resultPromise = sendCombinedNotification([
      { type: 'success', title: 'Test', message: 'msg' },
    ]);
    await vi.runAllTimersAsync();
    const result = await resultPromise;

    expect(result).toBe(false);
  });
});
