/**
 * DHLottery Authentication Module
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001
 *   task_id: TASK-002
 */

import type { AuthEnv } from '../types/auth.types';
import type { HttpClient } from '../types/http.types';
import { AuthenticationError } from '../utils/errors';

/**
 * DHLottery login endpoint
 */
const LOGIN_URL = 'https://www.dhlottery.co.kr/userSsl.do?method=login';

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
 * Authenticate user with DHLottery
 *
 * @param client - HTTP client with cookie management
 * @param env - Environment containing USER_ID and PASSWORD secrets
 * @throws {AuthenticationError} If login fails
 */
export async function login(client: HttpClient, env: AuthEnv): Promise<void> {
  // Prepare form data with credentials from secrets
  const formData = new URLSearchParams();
  formData.append('userId', env.USER_ID);
  formData.append('password', env.PASSWORD);

  try {
    // Send POST request to login endpoint
    const response = await client.fetch(LOGIN_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: formData.toString(),
    });

    // Check HTTP status
    if (response.status !== 200) {
      throw new AuthenticationError(
        `Login request failed with status ${response.status}`,
        'AUTH_HTTP_ERROR'
      );
    }

    // Parse response
    let responseData: DHLotteryLoginResponse;
    try {
      responseData = await response.json<DHLotteryLoginResponse>();
    } catch (_parseError) {
      throw new AuthenticationError('Failed to parse login response', 'AUTH_UNEXPECTED_RESPONSE');
    }

    // Validate response structure
    if (!responseData.result || !responseData.result.resultCode) {
      throw new AuthenticationError('Invalid login response format', 'AUTH_UNEXPECTED_RESPONSE');
    }

    // Check login result
    if (responseData.result.resultCode !== 'SUCCESS') {
      const errorMessage = responseData.result.resultMsg || 'Authentication failed';
      throw new AuthenticationError(errorMessage, 'AUTH_INVALID_CREDENTIALS');
    }

    // Login successful - cookies are automatically updated by HttpClient
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
