/**
 * DHLottery Authentication Tests
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001
 *   task_id: TASK-002
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { login } from "./auth";
import { createHttpClient } from "../client/http";
import { AuthenticationError } from "../utils/errors";
import type { AuthEnv } from "../types/auth.types";
import type { HttpResponse } from "../types/http.types";

describe("DHLottery Authentication", () => {
  let mockHttpClient: ReturnType<typeof createHttpClient>;
  let mockEnv: AuthEnv;

  beforeEach(() => {
    // Create mock HTTP client
    mockHttpClient = {
      cookies: {},
      fetch: vi.fn(),
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    } as unknown as ReturnType<typeof createHttpClient>;

    // Create mock environment with secrets
    mockEnv = {
      USER_ID: "testuser",
      PASSWORD: "testpass123",
    };
  });

  /**
   * TEST-AUTH-001: Should authenticate with valid credentials
   *
   * Criteria:
   * - POST request to /userSsl.do?method=login succeeds
   * - Response indicates successful login
   * - Session cookies are updated
   * - User is considered authenticated
   */
  describe("TEST-AUTH-001: Authenticate with valid credentials", () => {
    it("should successfully authenticate with valid credentials", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers({
          "set-cookie": "JSESSIONID=authenticated123; Path=/",
        }),
        text: async () =>
          JSON.stringify({
            result: {
              resultCode: "SUCCESS",
              resultMsg: "로그인 성공",
            },
          }),
        json: async () => ({
          result: {
            resultCode: "SUCCESS",
            resultMsg: "로그인 성공",
          },
        }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert
      expect(mockHttpClient.fetch).toHaveBeenCalledTimes(1);
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];

      // Verify URL
      expect(callArgs[0]).toContain("/userSsl.do?method=login");

      // Verify request method is POST
      expect(callArgs[1]?.method).toBe("POST");
    });

    it("should handle successful login with cookies", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert - should not throw
      expect(mockHttpClient.fetch).toHaveBeenCalled();
    });

    it("should parse successful login response", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () =>
          JSON.stringify({
            result: {
              resultCode: "SUCCESS",
            },
          }),
        json: async () => ({
          result: {
            resultCode: "SUCCESS",
          },
        }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert - should not throw
      await expect(login(mockHttpClient, mockEnv)).resolves.not.toThrow();
    });
  });

  /**
   * TEST-AUTH-002: Should reject invalid credentials
   *
   * Criteria:
   * - Login fails with invalid credentials
   * - AuthenticationError is thrown
   * - Error message describes the failure
   */
  describe("TEST-AUTH-002: Reject invalid credentials", () => {
    it("should throw AuthenticationError on invalid credentials", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () =>
          JSON.stringify({
            result: {
              resultCode: "FAIL",
              resultMsg: "아이디 또는 비밀번호가 일치하지 않습니다",
            },
          }),
        json: async () => ({
          result: {
            resultCode: "FAIL",
            resultMsg: "아이디 또는 비밀번호가 일치하지 않습니다",
          },
        }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(login(mockHttpClient, mockEnv)).rejects.toThrow(
        AuthenticationError,
      );
    });

    it("should include error message in thrown error", async () => {
      // Arrange
      const errorMessage = "인증 실패";
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () =>
          JSON.stringify({
            result: {
              resultCode: "FAIL",
              resultMsg: errorMessage,
            },
          }),
        json: async () => ({
          result: {
            resultCode: "FAIL",
            resultMsg: errorMessage,
          },
        }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(login(mockHttpClient, mockEnv)).rejects.toThrow(errorMessage);
    });

    it("should handle unexpected response format", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => "Invalid JSON",
        json: async () => {
          throw new Error("Invalid JSON");
        },
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(login(mockHttpClient, mockEnv)).rejects.toThrow(
        AuthenticationError,
      );
    });

    it("should handle non-200 HTTP status", async () => {
      // Arrange
      const mockResponse = {
        status: 500,
        statusText: "Internal Server Error",
        headers: new Headers(),
        text: async () => "Server Error",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(login(mockHttpClient, mockEnv)).rejects.toThrow(
        AuthenticationError,
      );
    });
  });

  /**
   * TEST-AUTH-003: Should use credentials from Secrets
   *
   * Criteria:
   * - USER_ID is read from env.USER_ID
   * - PASSWORD is read from env.PASSWORD
   * - No hardcoded credentials in code
   */
  describe("TEST-AUTH-003: Use credentials from Secrets", () => {
    it("should use USER_ID from environment", async () => {
      // Arrange
      const testUserId = "my-test-user";
      mockEnv.USER_ID = testUserId;

      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      const requestBody = callArgs[1]?.body as string;

      expect(requestBody).toContain(testUserId);
    });

    it("should use PASSWORD from environment", async () => {
      // Arrange
      const testPassword = "secret-password-123";
      mockEnv.PASSWORD = testPassword;

      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      const requestBody = callArgs[1]?.body as string;

      expect(requestBody).toContain(testPassword);
    });

    it("should work with different credentials", async () => {
      // Arrange
      mockEnv.USER_ID = "another-user";
      mockEnv.PASSWORD = "another-pass";

      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act & Assert
      await expect(login(mockHttpClient, mockEnv)).resolves.not.toThrow();
    });
  });

  /**
   * TEST-AUTH-004: Should send proper login request format
   *
   * Criteria:
   * - Content-Type is application/x-www-form-urlencoded
   * - Request method is POST
   * - Required form fields are included
   */
  describe("TEST-AUTH-004: Send proper login request format", () => {
    it("should send POST request with correct Content-Type", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];

      expect(callArgs[1]?.method).toBe("POST");
      expect(callArgs[1]?.headers?.["Content-Type"]).toBe(
        "application/x-www-form-urlencoded",
      );
    });

    it("should include userId and password in form data", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      const requestBody = callArgs[1]?.body as string;

      // Should be URL-encoded form data
      expect(requestBody).toContain("userId=");
      expect(requestBody).toContain("password=");
    });

    it("should properly URL-encode form data", async () => {
      // Arrange
      mockEnv.USER_ID = "test@email.com";
      mockEnv.PASSWORD = "p@ss w0rd!";

      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      const requestBody = callArgs[1]?.body as string;

      // URLSearchParams encodes @ and ! but uses + for spaces (not %20)
      expect(requestBody).toContain("test%40email.com");
      expect(requestBody).toContain("p%40ss+w0rd%21");
    });

    it("should send request to correct endpoint", async () => {
      // Arrange
      const mockResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => JSON.stringify({ result: { resultCode: "SUCCESS" } }),
        json: async () => ({ result: { resultCode: "SUCCESS" } }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValue(mockResponse);

      // Act
      await login(mockHttpClient, mockEnv);

      // Assert
      const callArgs = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      const url = callArgs[0];

      expect(url).toContain("dhlottery.co.kr");
      expect(url).toContain("/userSsl.do");
      expect(url).toContain("method=login");
    });
  });
});
