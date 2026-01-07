/**
 * HTTP Client Tests
 *
 * Trace:
 *   spec_id: SPEC-SESSION-001
 *   task_id: TASK-001
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { createHttpClient } from "./http";

/**
 * Create mock response with arrayBuffer support
 */
function createMockResponse(
	status: number,
	headers: Headers,
	body: string = "<html></html>"
): any {
	const mockBody = new TextEncoder().encode(body);
	return {
		status,
		statusText: status === 200 ? "OK" : "Error",
		headers,
		arrayBuffer: async () => mockBody.buffer,
	};
}

describe("HTTP Client - Session Management", () => {
	let mockFetch: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		// Mock global fetch
		mockFetch = vi.fn();
		global.fetch = mockFetch as any;
	});

	/**
	 * TEST-SESSION-001: Should capture session cookies from initial request
	 *
	 * Criteria:
	 * - Response contains Set-Cookie header
	 * - Cookies are stored in client state
	 * - JSESSIONID or similar session cookie is present
	 */
	describe("TEST-SESSION-001: Capture cookies from initial request", () => {
		it("should capture single cookie from Set-Cookie header", async () => {
			// Arrange
			const client = createHttpClient();
			const mockHeaders = new Headers({
				"set-cookie": "JSESSIONID=abc123; Path=/; HttpOnly",
			});

			mockFetch.mockResolvedValue(createMockResponse(200, mockHeaders));

			// Act
			await client.fetch("https://example.com");

			// Assert
			expect(client.cookies).toHaveProperty("JSESSIONID");
			expect(client.cookies.JSESSIONID).toBe("abc123");
		});

		it("should capture multiple cookies from Set-Cookie header", async () => {
			// Arrange
			const client = createHttpClient();
			const mockHeaders = new Headers();
			mockHeaders.append("set-cookie", "session=xyz789; Path=/");
			mockHeaders.append("set-cookie", "lang=ko; Path=/");

			mockFetch.mockResolvedValue(createMockResponse(200, mockHeaders));

			// Act
			await client.fetch("https://example.com");

			// Assert
			expect(client.cookies).toHaveProperty("session");
			expect(client.cookies).toHaveProperty("lang");
			expect(client.cookies.session).toBe("xyz789");
			expect(client.cookies.lang).toBe("ko");
		});

		it("should handle response with no Set-Cookie header", async () => {
			// Arrange
			const client = createHttpClient();
			const mockHeaders = new Headers({
				"content-type": "text/html",
			});

			mockFetch.mockResolvedValue(createMockResponse(200, mockHeaders));

			// Act
			await client.fetch("https://example.com");

			// Assert
			expect(Object.keys(client.cookies)).toHaveLength(0);
		});
	});

	/**
	 * TEST-SESSION-002: Should include cookies in subsequent requests
	 *
	 * Criteria:
	 * - Request headers contain Cookie field
	 * - Cookie values match stored session
	 */
	describe("TEST-SESSION-002: Include cookies in subsequent requests", () => {
		it("should include stored cookies in Cookie header", async () => {
			// Arrange
			const client = createHttpClient();
			const firstHeaders = new Headers({
				"set-cookie": "JSESSIONID=abc123; Path=/",
			});

			mockFetch.mockResolvedValueOnce(createMockResponse(200, firstHeaders));

			// First request to capture cookies
			await client.fetch("https://example.com");

			// Setup second request mock
			const secondHeaders = new Headers();
			mockFetch.mockResolvedValueOnce(createMockResponse(200, secondHeaders));

			// Act - Second request should include cookies
			await client.fetch("https://example.com/page");

			// Assert
			const secondCallOptions = mockFetch.mock.calls[1][1];
			expect(secondCallOptions.headers).toHaveProperty("Cookie");
			expect(secondCallOptions.headers.Cookie).toBe("JSESSIONID=abc123");
		});

		it("should include multiple cookies separated by semicolon", async () => {
			// Arrange
			const client = createHttpClient();
			const firstHeaders = new Headers();
			firstHeaders.append("set-cookie", "session=xyz; Path=/");
			firstHeaders.append("set-cookie", "lang=ko; Path=/");

			mockFetch.mockResolvedValueOnce(createMockResponse(200, firstHeaders));

			await client.fetch("https://example.com");

			mockFetch.mockResolvedValueOnce(createMockResponse(200, new Headers()));

			// Act
			await client.fetch("https://example.com/page");

			// Assert
			const secondCallOptions = mockFetch.mock.calls[1][1];
			expect(secondCallOptions.headers.Cookie).toContain("session=xyz");
			expect(secondCallOptions.headers.Cookie).toContain("lang=ko");
			expect(secondCallOptions.headers.Cookie).toMatch(/session=xyz; lang=ko/);
		});

		it("should not include Cookie header when no cookies stored", async () => {
			// Arrange
			const client = createHttpClient();
			mockFetch.mockResolvedValue(createMockResponse(200, new Headers()));

			// Act
			await client.fetch("https://example.com");

			// Assert
			const callOptions = mockFetch.mock.calls[0][1];
			expect(callOptions.headers).not.toHaveProperty("Cookie");
		});
	});

	/**
	 * TEST-SESSION-003: Should handle cookie updates during session
	 *
	 * Criteria:
	 * - New Set-Cookie headers update stored cookies
	 * - Old cookie values are replaced
	 * - Cookie store remains consistent
	 */
	describe("TEST-SESSION-003: Handle cookie updates", () => {
		it("should update existing cookie with new value", async () => {
			// Arrange
			const client = createHttpClient();

			// First request sets initial cookie
			mockFetch.mockResolvedValueOnce(
				createMockResponse(
					200,
					new Headers({
						"set-cookie": "session=initial; Path=/",
					})
				)
			);

			await client.fetch("https://example.com");
			expect(client.cookies.session).toBe("initial");

			// Second request updates the cookie
			mockFetch.mockResolvedValueOnce(
				createMockResponse(
					200,
					new Headers({
						"set-cookie": "session=updated; Path=/",
					})
				)
			);

			// Act
			await client.fetch("https://example.com/update");

			// Assert
			expect(client.cookies.session).toBe("updated");
			expect(Object.keys(client.cookies)).toHaveLength(1);
		});

		it("should add new cookies while keeping existing ones", async () => {
			// Arrange
			const client = createHttpClient();

			mockFetch.mockResolvedValueOnce(
				createMockResponse(
					200,
					new Headers({
						"set-cookie": "session=abc; Path=/",
					})
				)
			);

			await client.fetch("https://example.com");

			mockFetch.mockResolvedValueOnce(
				createMockResponse(
					200,
					new Headers({
						"set-cookie": "lang=ko; Path=/",
					})
				)
			);

			// Act
			await client.fetch("https://example.com/login");

			// Assert
			expect(client.cookies.session).toBe("abc");
			expect(client.cookies.lang).toBe("ko");
			expect(Object.keys(client.cookies)).toHaveLength(2);
		});

		it("should handle mixed cookie operations (update + add)", async () => {
			// Arrange
			const client = createHttpClient();

			mockFetch.mockResolvedValueOnce(
				createMockResponse(
					200,
					new Headers({
						"set-cookie": "session=v1; Path=/",
					})
				)
			);

			await client.fetch("https://example.com");

			const updateHeaders = new Headers();
			updateHeaders.append("set-cookie", "session=v2; Path=/");
			updateHeaders.append("set-cookie", "user=john; Path=/");

			mockFetch.mockResolvedValueOnce(createMockResponse(200, updateHeaders));

			// Act
			await client.fetch("https://example.com/auth");

			// Assert
			expect(client.cookies.session).toBe("v2"); // updated
			expect(client.cookies.user).toBe("john"); // added
			expect(Object.keys(client.cookies)).toHaveLength(2);
		});
	});

	describe("Cookie utility methods", () => {
		it("getCookieHeader should return formatted cookie string", () => {
			// Arrange
			const client = createHttpClient();
			mockFetch.mockResolvedValue(createMockResponse(200, new Headers()));

			// Manually set cookies for testing
			client.cookies.session = "abc";
			client.cookies.lang = "ko";

			// Act
			const cookieHeader = client.getCookieHeader();

			// Assert
			expect(cookieHeader).toContain("session=abc");
			expect(cookieHeader).toContain("lang=ko");
		});

		it("clearCookies should remove all stored cookies", async () => {
			// Arrange
			const client = createHttpClient();
			mockFetch.mockResolvedValue(
				createMockResponse(
					200,
					new Headers({
						"set-cookie": "session=xyz; Path=/",
					})
				)
			);

			await client.fetch("https://example.com");
			expect(Object.keys(client.cookies)).toHaveLength(1);

			// Act
			client.clearCookies();

			// Assert
			expect(Object.keys(client.cookies)).toHaveLength(0);
		});
	});
});
