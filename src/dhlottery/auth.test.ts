/**
 * DHLottery Authentication Tests
 *
 * Tests for RSA-encrypted authentication flow (updated 2026-01):
 * 1. Session initialization (GET /login) - acquire DHJSESSIONID cookie
 * 2. RSA key fetch (GET /login/selectRsaModulus.do) - get public key
 * 3. Login (POST /login/securityLoginCheck.do) - submit encrypted credentials
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001, SPEC-AUTH-RSA-001
 *   task_id: TASK-002, TASK-AUTH-RSA-001
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { login } from "./auth";
import { createHttpClient } from "../client/http";
import { AuthenticationError } from "../utils/errors";
import type { HttpResponse } from "../types";

describe("DHLottery Authentication", () => {
  let mockHttpClient: ReturnType<typeof createHttpClient>;

  beforeEach(() => {
    // Mock process.env
    vi.stubEnv('USER_ID', 'testuser');
    vi.stubEnv('PASSWORD', 'testpass123');

    // Create mock HTTP client
    mockHttpClient = {
      cookies: {
        // Pre-set DHJSESSIONID to satisfy initSession verification
        DHJSESSIONID: 'mock-session-id',
      },
      fetch: vi.fn(),
      getCookieHeader: vi.fn(),
      clearCookies: vi.fn(),
    } as unknown as ReturnType<typeof createHttpClient>;
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  /**
   * Helper function to create mock responses for new RSA-based auth flow
   */
  const createMockResponses = (loginSuccess: boolean, errorMessage?: string) => {
    // Step 1: Session init response (GET /login)
    const sessionInitResponse = {
      status: 200,
      statusText: "OK",
      headers: new Headers(),
      text: async () => "<!DOCTYPE html>...",
      json: async () => ({}),
    } as unknown as HttpResponse;

    // Step 2: RSA key response (GET /login/selectRsaModulus.do)
    const rsaKeyResponse = {
      status: 200,
      statusText: "OK",
      headers: new Headers({ 'content-type': 'application/json' }),
      text: async () => JSON.stringify({
        code: "0000",
        msg: "success",
        data: {
          // Valid 2048-bit RSA modulus (512 hex chars)
          rsaModulus: "9e3076fbc2019ec646dbe2a6d0d22ba2241d85ea363b77f583a9ca0646fae3032d5100a41f67794ca5ff80fc360f9429a0a075be9ef3a1bca4e8bea41915a4491da0e4afdc4081bf3500a502717d913638daaf4883c25561c4f5937fe86478dca94f65624a7de907a7f363994593a69346d90e8f8f8a709cb61936b8bad07d4e29343f11fafa62245662480a4b8206858e4743aea6f410f1d1ad85dafffe77dd2301e0dd5a5b0d5c7f63795cbdc8cd7bc4c90c9f0e6e534227c0f696957badad1b5af3f6722ee99d166f5cf574ae2524763e54590cbd4c9a57ab4d6851bb1870d45bc7439b7805e4be7ae7eca2ff66e7f3186fd21adda6f76efb9b5af8fa01e9",
          publicExponent: "10001",
        },
      }),
      json: async () => ({
        code: "0000",
        msg: "success",
        data: {
          rsaModulus: "9e3076fbc2019ec646dbe2a6d0d22ba2241d85ea363b77f583a9ca0646fae3032d5100a41f67794ca5ff80fc360f9429a0a075be9ef3a1bca4e8bea41915a4491da0e4afdc4081bf3500a502717d913638daaf4883c25561c4f5937fe86478dca94f65624a7de907a7f363994593a69346d90e8f8f8a709cb61936b8bad07d4e29343f11fafa62245662480a4b8206858e4743aea6f410f1d1ad85dafffe77dd2301e0dd5a5b0d5c7f63795cbdc8cd7bc4c90c9f0e6e534227c0f696957badad1b5af3f6722ee99d166f5cf574ae2524763e54590cbd4c9a57ab4d6851bb1870d45bc7439b7805e4be7ae7eca2ff66e7f3186fd21adda6f76efb9b5af8fa01e9",
          publicExponent: "10001",
        },
      }),
    } as unknown as HttpResponse;

    // Step 3: Login response (POST /login/securityLoginCheck.do)
    const loginResponseHtml = loginSuccess
      ? `<script>const isLoggedIn = true;</script>`
      : `<script>const isLoggedIn = false; const errorMessage = '${errorMessage || "아이디 또는 비밀번호가 일치하지 않습니다."}';</script>`;

    const loginResponse = {
      status: 200,
      statusText: "OK",
      headers: new Headers({ 'content-type': 'text/html;charset=UTF-8' }),
      text: async () => loginResponseHtml,
      json: async () => ({}),
    } as unknown as HttpResponse;

    return { sessionInitResponse, rsaKeyResponse, loginResponse };
  };

  /**
   * TEST-AUTH-001: Should authenticate with valid credentials
   *
   * Criteria:
   * - Session init (GET /login) succeeds
   * - RSA key fetch (GET /login/selectRsaModulus.do) succeeds
   * - Login (POST /login/securityLoginCheck.do) succeeds
   * - Session cookies are updated (DHJSESSIONID)
   */
  describe("TEST-AUTH-001: Authenticate with valid credentials", () => {
    it("should successfully authenticate with valid credentials", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - should be called 3 times (session init + RSA key + login)
      expect(mockHttpClient.fetch).toHaveBeenCalledTimes(3);

      // Verify first call is session init
      const sessionCall = vi.mocked(mockHttpClient.fetch).mock.calls[0];
      expect(sessionCall[0]).toBe("https://dhlottery.co.kr/login");

      // Verify second call is RSA key fetch
      const rsaKeyCall = vi.mocked(mockHttpClient.fetch).mock.calls[1];
      expect(rsaKeyCall[0]).toBe("https://dhlottery.co.kr/login/selectRsaModulus.do");

      // Verify third call is login
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];
      expect(loginCall[0]).toBe("https://dhlottery.co.kr/login/securityLoginCheck.do");
      expect(loginCall[1]?.method).toBe("POST");
    });

    it("should handle successful login with cookies", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - should not throw
      expect(mockHttpClient.fetch).toHaveBeenCalledTimes(3);
    });

    it("should parse successful login response", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act & Assert - should not throw
      await expect(login(mockHttpClient)).resolves.not.toThrow();
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
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(
        false,
        "아이디 또는 비밀번호가 일치하지 않습니다."
      );

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });

    it("should include error message in thrown error", async () => {
      // Arrange
      const errorMessage = "인증 실패";
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(
        false,
        errorMessage
      );

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(errorMessage);
    });

    it("should handle unexpected response format", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse } = createMockResponses(true);

      const invalidLoginResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers(),
        text: async () => "Unexpected response without isLoggedIn",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(invalidLoginResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });

    it("should handle non-200 HTTP status on login", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse } = createMockResponses(true);

      const errorLoginResponse = {
        status: 500,
        statusText: "Internal Server Error",
        headers: new Headers(),
        text: async () => "Server Error",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(errorLoginResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });

    it("should handle non-200 HTTP status on session init", async () => {
      // Arrange
      const errorSessionInitResponse = {
        status: 500,
        statusText: "Internal Server Error",
        headers: new Headers(),
        text: async () => "Server Error",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch).mockResolvedValueOnce(errorSessionInitResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });
  });

  /**
   * TEST-AUTH-003: Should use credentials from Secrets
   *
   * Criteria:
   * - USER_ID is read from env.USER_ID
   * - PASSWORD is read from env.PASSWORD
   * - Credentials are RSA encrypted before sending
   */
  describe("TEST-AUTH-003: Use credentials from Secrets", () => {
    it("should use USER_ID from environment (in inpUserId field)", async () => {
      // Arrange
      const testUserId = "my-test-user";
      vi.stubEnv('USER_ID', testUserId);

      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - check third call (login request)
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];
      const requestBody = loginCall[1]?.body as string;

      // inpUserId contains plain text userId
      expect(requestBody).toContain(`inpUserId=${testUserId}`);
    });

    it("should encrypt PASSWORD with RSA", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - check third call (login request)
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];
      const requestBody = loginCall[1]?.body as string;

      // userPswdEncn should contain RSA encrypted password (hex string, not plain password)
      expect(requestBody).toContain("userPswdEncn=");
      expect(requestBody).not.toContain("userPswdEncn=testpass123");
    });

    it("should work with different credentials", async () => {
      // Arrange
      vi.stubEnv('USER_ID', 'another-user');
      vi.stubEnv('PASSWORD', 'another-pass');

      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).resolves.not.toThrow();
    });
  });

  /**
   * TEST-AUTH-004: Should send proper login request format
   *
   * Criteria:
   * - Content-Type is application/x-www-form-urlencoded
   * - Request method is POST
   * - Required form fields are included (userId, userPswdEncn, inpUserId)
   */
  describe("TEST-AUTH-004: Send proper login request format", () => {
    it("should send POST request with correct Content-Type", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - check third call (login request)
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];

      expect(loginCall[1]?.method).toBe("POST");
      expect(loginCall[1]?.headers?.["Content-Type"]).toBe(
        "application/x-www-form-urlencoded",
      );
    });

    it("should include required browser-like headers", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - check third call (login request)
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];
      const headers = loginCall[1]?.headers;

      expect(headers?.["User-Agent"]).toBeDefined();
      expect(headers?.["Origin"]).toBe("https://dhlottery.co.kr");
      expect(headers?.["Referer"]).toBe("https://dhlottery.co.kr/login");
    });

    it("should include all required form parameters", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - check third call (login request)
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];
      const requestBody = loginCall[1]?.body as string;

      // New form fields for RSA-based auth
      expect(requestBody).toContain("userId=");       // RSA encrypted userId
      expect(requestBody).toContain("userPswdEncn="); // RSA encrypted password
      expect(requestBody).toContain("inpUserId=");    // Plain text userId
    });

    it("should send RSA encrypted credentials (not URL-encoded plain text)", async () => {
      // Arrange
      vi.stubEnv('USER_ID', 'test@email.com');
      vi.stubEnv('PASSWORD', 'p@ss w0rd!');

      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - check third call (login request)
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];
      const requestBody = loginCall[1]?.body as string;

      // inpUserId should be URL-encoded plain text
      expect(requestBody).toContain("inpUserId=test%40email.com");

      // userId and userPswdEncn should be RSA encrypted (hex strings)
      // They should NOT contain the plain password
      expect(requestBody).not.toContain("p%40ss");
    });

    it("should send request to correct endpoint", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - check third call (login request)
      const loginCall = vi.mocked(mockHttpClient.fetch).mock.calls[2];
      const url = loginCall[0];

      expect(url).toBe("https://dhlottery.co.kr/login/securityLoginCheck.do");
    });
  });

  /**
   * TEST-AUTH-005: Treat 302 redirect as success with manual redirects
   *
   * Criteria:
   * - Login response returns 302 redirect
   * - Response body may be empty
   * - Authentication succeeds without userId cookie
   */
  describe("TEST-AUTH-005: Treat 302 redirect as success with manual redirects", () => {
    it("should succeed when login returns 302 redirect", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse } = createMockResponses(true);

      const redirectLoginResponse = {
        status: 302,
        statusText: "Found",
        headers: new Headers({ location: "/login/loginSuccess.do" }),
        text: async () => "",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(redirectLoginResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).resolves.not.toThrow();
    });
  });

  /**
   * TEST-AUTH-006: Reject non-success 302 redirect
   *
   * Criteria:
   * - Login response returns 302 redirect
   * - Location header does not indicate login success
   * - Authentication fails
   */
  describe("TEST-AUTH-006: Reject non-success 302 redirect", () => {
    it("should fail when login returns 302 to non-success location", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse } = createMockResponses(true);

      const redirectLoginResponse = {
        status: 302,
        statusText: "Found",
        headers: new Headers({ location: "/login/loginFail.do" }),
        text: async () => "",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(redirectLoginResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });
  });

  /**
   * TEST-AUTH-RSA-001: RSA key fetch
   *
   * Criteria:
   * - RSA modulus and exponent are fetched before login
   * - Invalid RSA response throws error
   */
  describe("TEST-AUTH-RSA-001: RSA key fetch", () => {
    it("should handle RSA key fetch failure", async () => {
      // Arrange
      const { sessionInitResponse } = createMockResponses(true);

      const errorRsaResponse = {
        status: 500,
        statusText: "Internal Server Error",
        headers: new Headers(),
        text: async () => "Server Error",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(errorRsaResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });

    it("should handle invalid RSA key response format", async () => {
      // Arrange
      const { sessionInitResponse } = createMockResponses(true);

      const invalidRsaResponse = {
        status: 200,
        statusText: "OK",
        headers: new Headers({ 'content-type': 'application/json' }),
        text: async () => JSON.stringify({ data: {} }),
        json: async () => ({ data: {} }),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(invalidRsaResponse);

      // Act & Assert
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });
  });

  /**
   * TEST-REFACTOR-P2-LOG-002: Conditional logging with DEBUG flag
   *
   * Criteria:
   * - console.log calls are wrapped with if (DEBUG) condition
   * - When DEBUG is false, console.log should not be called
   * - Error handling still works correctly
   */
  describe("TEST-REFACTOR-P2-LOG-002: Conditional logging with DEBUG flag", () => {
    let consoleLogSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
      consoleLogSpy = vi.spyOn(console, 'log');
    });

    it("should skip console.log calls when DEBUG is false", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse, loginResponse } = createMockResponses(true);

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(loginResponse);

      // Act
      await login(mockHttpClient);

      // Assert - console.log should NOT be called since DEBUG is false
      expect(consoleLogSpy).not.toHaveBeenCalled();
    });

    it("should not affect error handling when DEBUG is false", async () => {
      // Arrange
      const { sessionInitResponse, rsaKeyResponse } = createMockResponses(true);

      const errorResponse = {
        status: 500,
        statusText: "Internal Server Error",
        headers: new Headers(),
        text: async () => "Server Error",
        json: async () => ({}),
      } as unknown as HttpResponse;

      vi.mocked(mockHttpClient.fetch)
        .mockResolvedValueOnce(sessionInitResponse)
        .mockResolvedValueOnce(rsaKeyResponse)
        .mockResolvedValueOnce(errorResponse);

      // Act & Assert - error should still be thrown
      await expect(login(mockHttpClient)).rejects.toThrow(AuthenticationError);
    });
  });

  /**
   * TEST-REFACTOR-P0-AUTH-001: Ensure auth redirect comment reflects manual behavior
   *
   * Criteria:
   * - Comment references manual redirects (redirect: 'manual')
   */
  describe("TEST-REFACTOR-P0-AUTH-001: redirect comment clarity", () => {
    it("should mention redirect: 'manual' in auth login flow comment", () => {
      const fs = require("fs");
      const path = require("path");
      const sourceFile = path.join(__dirname, "auth.ts");
      const content = fs.readFileSync(sourceFile, "utf-8");

      expect(content).toMatch(/redirect:\s*'manual'/);
    });
  });
});
