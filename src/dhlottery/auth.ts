/**
 * DHLottery Authentication Module
 *
 * Implements two-phase authentication flow:
 * 1. Session initialization: GET gameResult.do to acquire JSESSIONID cookie
 * 2. User login: POST credentials to userSsl.do with browser-like headers
 *
 * Session URL strategy documented in SPEC-REFACTOR-P2-SESSION-001:
 * - Current implementation uses gameResult.do (from reference implementation)
 * - Alternative common.do also works (both are functionally equivalent)
 * - Rationale: Preserving reference implementation choice, changing carries low benefit/high risk
 * - See memory.md (TASK-013) for historical context and design evolution
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001, SPEC-REFACTOR-P2-LOG-001, SPEC-REFACTOR-P2-SESSION-001
 *   task_id: TASK-002, TASK-011, TASK-REFACTOR-P2-002, TASK-REFACTOR-P2-004
 */

import { DEBUG, USER_AGENT } from '../constants';
import type { AuthEnv, HttpClient } from '../types';
import { AuthenticationError } from '../utils/errors';

/**
 * DHLottery endpoints
 *
 * SESSION_INIT_URL: gameResult.do vs common.do
 *   - Uses gameResult.do (winning results page) with wiselog=H_C_1_1 parameter
 *   - Both gameResult.do and common.do return HTTP 200 with Set-Cookie: JSESSIONID
 *   - gameResult.do chosen from reference implementation (n8n workflow)
 *   - common.do would be more semantically appropriate (home page) but gameResult.do works
 *   - Design decision: Keep current implementation (stable, proven, no functional gain from changing)
 *   - Future refactoring: common.do is valid drop-in replacement if needed
 *
 * RETURN_URL: Used in login form to specify where browser should navigate after login
 *   - Points to common.do?method=main (home page)
 *   - We do NOT follow this redirect to preserve session cookies
 *   - getAccountInfo() fetches this URL later to stabilize session
 */
const SESSION_INIT_URL = 'https://dhlottery.co.kr/gameResult.do?method=byWin&wiselog=H_C_1_1';
const LOGIN_URL = 'https://www.dhlottery.co.kr/userSsl.do?method=login';
const RETURN_URL = 'https://dhlottery.co.kr/common.do?method=main';

/**
 * Login response structure from DHLottery
 */
interface DHLotteryLoginResponse {
  result?: {
    resultCode: string;
    resultMsg?: string;
  };
}

/**
 * Initialize session by requesting gameResult.do to acquire JSESSIONID cookie
 *
 * Design notes:
 *   - MUST be called before login() to establish session with JSESSIONID
 *   - Uses gameResult.do (winning results page) - derived from reference implementation
 *   - Returns HTTP 200 with Set-Cookie: JSESSIONID header
 *   - Alternative: common.do?method=main also works (verified in SPEC-REFACTOR-P2-SESSION-001)
 *   - Verification: Throws error if JSESSIONID is not set after the request
 *
 * Historical context:
 *   - TASK-013 (2025-12-18): Implemented two-phase authentication flow
 *   - Initial spec mentioned common.do, but implementation uses gameResult.do
 *   - gameResult.do is stable, tested, and proven to work in production
 *   - No functional advantage to changing, so kept for consistency
 *
 * @param client - HTTP client with cookie management
 * @throws {AuthenticationError} If session initialization fails or JSESSIONID is not set
 *
 * Trace: SPEC-REFACTOR-P2-SESSION-001 (session URL strategy investigation)
 */
async function initSession(client: HttpClient): Promise<void> {
  try {
    const response = await client.fetch(SESSION_INIT_URL, {
      method: 'GET',
      headers: {
        'Accept-Charset': 'UTF-8',
        'User-Agent': USER_AGENT,
      },
    });

    // Accept 200 OK
    if (response.status !== 200) {
      throw new AuthenticationError(
        `Session initialization failed with status ${response.status}`,
        'AUTH_SESSION_INIT_ERROR'
      );
    }

    if (DEBUG) {
      console.log(
        JSON.stringify({
          level: 'debug',
          module: 'auth',
          message: 'Session initialized',
          cookies: client.cookies,
          status: response.status,
        })
      );
    }

    // Verify JSESSIONID is set (critical for login)
    if (!client.cookies.JSESSIONID) {
      throw new AuthenticationError(
        'JSESSIONID cookie was not set during initialization',
        'AUTH_SESSION_INIT_ERROR'
      );
    }
  } catch (error) {
    if (error instanceof AuthenticationError) {
      throw error;
    }
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    throw new AuthenticationError(
      `Session initialization failed: ${errorMessage}`,
      'AUTH_NETWORK_ERROR'
    );
  }
}

/**
 * Authenticate user with DHLottery
 *
 * @param client - HTTP client with cookie management
 * @param env - Environment containing USER_ID and PASSWORD secrets
 * @throws {AuthenticationError} If login fails
 */
export async function login(client: HttpClient, env: AuthEnv): Promise<void> {
  // Step 1: Initialize session to get initial cookies
  await initSession(client);
  // Step 2: Prepare form data with credentials and required parameters
  const formData = new URLSearchParams();
  formData.append('returnUrl', RETURN_URL);
  formData.append('userId', env.USER_ID);
  formData.append('password', env.PASSWORD);
  formData.append('checkSave', 'off');
  formData.append('newsEventYn', '');

  try {
    // Step 3: Send POST request to login endpoint with browser-like headers
    const response = await client.fetch(LOGIN_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
        'User-Agent': USER_AGENT,
        'X-Requested-With': 'XMLHttpRequest',
        'sec-ch-ua-mobile': '?0',
        Referer: 'https://dhlottery.co.kr',
        'Sec-Fetch-Site': 'same-site',
        Connection: 'keep-alive',
        'sec-ch-ua': '" Not;A Brand";v="99", "Google Chrome";v="91", "Chromium";v="91"',
      },
      body: formData.toString(),
    });

    // Accept 200 OK or 3xx redirect
    if (response.status !== 200 && (response.status < 300 || response.status >= 400)) {
      throw new AuthenticationError(
        `Login request failed with status ${response.status}`,
        'AUTH_HTTP_ERROR'
      );
    }

    // Debug: Check Set-Cookie headers from login response
    if (DEBUG) {
      console.log(
        JSON.stringify({
          level: 'debug',
          module: 'auth',
          message: 'Login response received',
          status: response.status,
          setCookie: response.headers.getSetCookie(),
          cookies: client.cookies,
        })
      );
    }

    // 302 redirect = success
    if (response.status >= 300 && response.status < 400) {
      const location = response.headers.get('location');
      if (DEBUG) {
        console.log(
          JSON.stringify({
            level: 'info',
            module: 'auth',
            message: 'Login successful - received redirect',
            status: response.status,
            location,
            cookies: client.cookies,
          })
        );
      }

      // Do NOT follow the redirect manually.
      // n8n workflow uses the original session cookie for subsequent requests.
      // Following loginResult might reset the session or require complex handling.
      // We will let getAccountInfo() fetch the main page, which serves as the returnUrl visit.
      return;
    }

    // Parse response body with euc-kr encoding (server sends euc-kr)
    const responseText = await response.text('euc-kr');
    const contentType = response.headers.get('content-type') || '';

    // DHLottery returns HTML redirect page on successful login, JSON on failure
    if (contentType.includes('text/html')) {
      // HTML response indicates successful login with redirect
      // The response contains JavaScript that redirects to returnUrl or /common.do?method=main
      // Verify it's a success page by checking for goNextPage function
      if (responseText.includes('goNextPage')) {
        if (DEBUG) {
          console.log(
            JSON.stringify({
              level: 'info',
              module: 'auth',
              message: 'Login successful - session cookies established',
              cookies: client.cookies,
              cookieHeader: client.getCookieHeader(),
            })
          );
        }

        // DO NOT follow redirect - session cookies (WMONID, JSESSIONID) are already set
        // Following redirect may cause session loss
        return;
      }
      // HTML but not the expected success page
      throw new AuthenticationError(
        'Unexpected HTML response from login endpoint',
        'AUTH_UNEXPECTED_RESPONSE'
      );
    }

    // Try to parse as JSON (error responses are typically JSON)
    let responseData: DHLotteryLoginResponse;
    try {
      responseData = JSON.parse(responseText) as DHLotteryLoginResponse;
    } catch (_parseError) {
      throw new AuthenticationError(
        'Login response is neither valid HTML nor JSON',
        'AUTH_UNEXPECTED_RESPONSE'
      );
    }

    // Validate JSON response structure
    if (!responseData.result || !responseData.result.resultCode) {
      throw new AuthenticationError('Invalid login response format', 'AUTH_UNEXPECTED_RESPONSE');
    }

    // Check login result (JSON response typically indicates failure)
    if (responseData.result.resultCode !== 'SUCCESS') {
      const errorMessage = responseData.result.resultMsg || 'Authentication failed';
      throw new AuthenticationError(errorMessage, 'AUTH_INVALID_CREDENTIALS');
    }

    // JSON success response (if it exists)
    if (DEBUG) {
      console.log(
        JSON.stringify({
          level: 'info',
          module: 'auth',
          message: 'Login successful (JSON response)',
        })
      );
    }
  } catch (error) {
    // Re-throw AuthenticationError as-is
    if (error instanceof AuthenticationError) {
      throw error;
    }

    // Wrap other errors in AuthenticationError
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    throw new AuthenticationError(`Login failed: ${errorMessage}`, 'AUTH_NETWORK_ERROR');
  }
}
