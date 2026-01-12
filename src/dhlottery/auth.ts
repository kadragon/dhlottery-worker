/**
 * DHLottery Authentication Module
 *
 * Implements RSA-encrypted authentication flow (updated 2026-01):
 * 1. Session initialization: GET /login to acquire DHJSESSIONID cookie
 * 2. RSA key fetch: GET /login/selectRsaModulus.do to get public key
 * 3. User login: POST RSA-encrypted credentials to /login/securityLoginCheck.do
 *
 * Historical note:
 * - Pre-2026: Used userSsl.do with plain text credentials
 * - Post-2026: Uses securityLoginCheck.do with RSA PKCS#1 v1.5 encryption
 *
 * Trace:
 *   spec_id: SPEC-AUTH-001, SPEC-AUTH-RSA-001
 *   task_id: TASK-002, TASK-AUTH-RSA-001
 */

import forge from 'node-forge';
import { USER_AGENT } from '../constants';
import type { HttpClient } from '../types';
import { getEnv } from '../utils/env';
import { AuthenticationError, wrapAuthError } from '../utils/errors';
import { logger } from '../utils/logger';

/**
 * DHLottery endpoints
 */
const LOGIN_PAGE_URL = 'https://dhlottery.co.kr/login';
const RSA_MODULUS_URL = 'https://dhlottery.co.kr/login/selectRsaModulus.do';
const LOGIN_URL = 'https://dhlottery.co.kr/login/securityLoginCheck.do';

/**
 * RSA Modulus response from DHLottery
 */
interface RsaModulusResponse {
  code: string;
  msg: string;
  data: {
    rsaModulus: string;
    publicExponent: string;
  };
}

/**
 * Encrypt text using RSA PKCS#1 v1.5 padding
 *
 * Uses node-forge for RSA encryption as it produces output compatible
 * with DHLottery's JavaScript RSA implementation (jsbn.js based).
 * Node.js native crypto module produces different ciphertext that
 * DHLottery's server does not accept.
 *
 * @param text - Plain text to encrypt
 * @param modulus - RSA modulus as hex string
 * @param exponent - RSA public exponent as hex string
 * @returns Encrypted text as hex string
 */
function rsaEncrypt(text: string, modulus: string, exponent: string): string {
  // Create BigInteger from hex strings for modulus and exponent
  const n = new forge.jsbn.BigInteger(modulus, 16);
  const e = new forge.jsbn.BigInteger(exponent, 16);

  // Create RSA public key from modulus and exponent
  const publicKey = forge.pki.setRsaPublicKey(n, e);

  // Encrypt with PKCS#1 v1.5 padding (matches DHLottery's RSA implementation)
  const encrypted = publicKey.encrypt(text, 'RSAES-PKCS1-V1_5');

  // Convert encrypted bytes to hex string
  return forge.util.bytesToHex(encrypted);
}

/**
 * Initialize session by requesting login page to acquire DHJSESSIONID cookie
 *
 * Handles 301/302 redirects manually since HttpClient uses redirect: 'manual'.
 * Follows up to 5 redirects to prevent infinite loops.
 *
 * @param client - HTTP client with cookie management
 * @throws {AuthenticationError} If session initialization fails
 */
async function initSession(client: HttpClient): Promise<void> {
  const MAX_REDIRECTS = 5;

  try {
    let currentUrl = LOGIN_PAGE_URL;
    let redirectCount = 0;

    while (redirectCount < MAX_REDIRECTS) {
      const response = await client.fetch(currentUrl, {
        method: 'GET',
        headers: {
          'Accept-Charset': 'UTF-8',
          'User-Agent': USER_AGENT,
        },
      });

      // Success - got the page
      if (response.status === 200) {
        logger.debug('Session initialized', {
          module: 'auth',
          cookies: client.cookies,
          status: response.status,
        });

        // Verify DHJSESSIONID is set (new cookie name since 2026)
        if (!client.cookies.DHJSESSIONID) {
          throw new AuthenticationError(
            'DHJSESSIONID cookie was not set during initialization',
            'AUTH_SESSION_INIT_ERROR'
          );
        }
        return;
      }

      // Handle redirects (301, 302, 303, 307, 308)
      if (response.status >= 300 && response.status < 400) {
        const location = response.headers.get('location');
        if (!location) {
          throw new AuthenticationError(
            `Session initialization redirect without Location header (status ${response.status})`,
            'AUTH_SESSION_INIT_ERROR'
          );
        }

        // Resolve relative URL to absolute
        currentUrl = new URL(location, currentUrl).toString();
        redirectCount++;

        logger.debug('Session init redirect', {
          module: 'auth',
          status: response.status,
          location: currentUrl,
          redirectCount,
        });
        continue;
      }

      // Non-success, non-redirect status
      throw new AuthenticationError(
        `Session initialization failed with status ${response.status}`,
        'AUTH_SESSION_INIT_ERROR'
      );
    }

    // Exceeded max redirects
    throw new AuthenticationError(
      `Session initialization exceeded maximum redirects (${MAX_REDIRECTS})`,
      'AUTH_SESSION_INIT_ERROR'
    );
  } catch (error) {
    throw wrapAuthError(error, 'Session initialization');
  }
}

/**
 * Fetch RSA public key from DHLottery
 *
 * @param client - HTTP client with cookie management
 * @returns RSA modulus and exponent
 * @throws {AuthenticationError} If RSA key fetch fails
 */
async function fetchRsaKey(client: HttpClient): Promise<{ modulus: string; exponent: string }> {
  try {
    const response = await client.fetch(RSA_MODULUS_URL, {
      method: 'GET',
      headers: {
        Accept: 'application/json, text/javascript, */*; q=0.01',
        'Content-Type': 'application/json;charset=UTF-8',
        'User-Agent': USER_AGENT,
        'X-Requested-With': 'XMLHttpRequest',
        Referer: LOGIN_PAGE_URL,
        ajax: 'true',
      },
    });

    if (response.status !== 200) {
      throw new AuthenticationError(
        `RSA key fetch failed with status ${response.status}`,
        'AUTH_RSA_KEY_ERROR'
      );
    }

    const data = await response.json<RsaModulusResponse>();

    if (!data.data?.rsaModulus || !data.data?.publicExponent) {
      throw new AuthenticationError('Invalid RSA key response format', 'AUTH_RSA_KEY_ERROR');
    }

    logger.debug('RSA key fetched', {
      module: 'auth',
      modulusLength: data.data.rsaModulus.length,
    });

    return {
      modulus: data.data.rsaModulus,
      exponent: data.data.publicExponent,
    };
  } catch (error) {
    throw wrapAuthError(error, 'RSA key fetch');
  }
}

/**
 * Authenticate user with DHLottery
 *
 * Authentication flow:
 * 1. Initialize session (GET /login) to acquire DHJSESSIONID cookie
 * 2. Fetch RSA public key (GET /login/selectRsaModulus.do)
 * 3. Encrypt userId and password with RSA PKCS#1 v1.5
 * 4. Submit login form (POST /login/securityLoginCheck.do)
 *
 * @param client - HTTP client with cookie management
 * @throws {AuthenticationError} If login fails
 */
export async function login(client: HttpClient): Promise<void> {
  // Step 1: Initialize session to get DHJSESSIONID cookie
  await initSession(client);

  // Step 2: Fetch RSA public key
  const rsaKey = await fetchRsaKey(client);

  // Step 3: Get credentials from environment
  const userId = getEnv('USER_ID');
  const password = getEnv('PASSWORD');

  // Step 4: Encrypt credentials with RSA
  const encryptedUserId = rsaEncrypt(userId, rsaKey.modulus, rsaKey.exponent);
  const encryptedPassword = rsaEncrypt(password, rsaKey.modulus, rsaKey.exponent);

  // Step 5: Prepare form data with encrypted credentials
  const formData = new URLSearchParams();
  formData.append('userId', encryptedUserId);
  formData.append('userPswdEncn', encryptedPassword);
  formData.append('inpUserId', userId);

  try {
    // Step 6: Send POST request to login endpoint
    const response = await client.fetch(LOGIN_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'User-Agent': USER_AGENT,
        Origin: 'https://dhlottery.co.kr',
        Referer: LOGIN_PAGE_URL,
        'Upgrade-Insecure-Requests': '1',
        'Cache-Control': 'max-age=0',
      },
      body: formData.toString(),
    });

    logger.debug('Login response received', {
      module: 'auth',
      status: response.status,
      cookies: client.cookies,
    });

    // Manual redirects: successful login typically returns 302 with empty body.
    if (response.status >= 300 && response.status < 400) {
      const location = response.headers.get('location') ?? '';
      if (location.includes('loginSuccess.do')) {
        logger.debug('Login successful - redirect received', {
          module: 'auth',
          status: response.status,
          location,
        });
        return;
      }
    }

    // Check for successful login via redirect (302 to loginSuccess.do)
    // HttpClient uses redirect: 'manual', so we may not follow the redirect here.
    // A userId cookie indicates successful login.
    if (client.cookies.userId) {
      logger.debug('Login successful - session established', {
        module: 'auth',
        cookies: client.cookies,
      });
      return;
    }

    // Parse response body for additional checks
    const responseText = await response.text('utf-8');

    // Check for login success indicator in HTML response
    // Successful login may also show isLoggedIn = true after redirect
    if (responseText.includes('isLoggedIn = true')) {
      logger.debug('Login successful - session established', {
        module: 'auth',
        cookies: client.cookies,
      });
      return;
    }

    // Check for error message in response
    // Failed login returns HTML with error message displayed via $.alert()
    const errorMatch = responseText.match(/\$\.alert\(['"]([^'"]+)['"]\)/);
    if (errorMatch) {
      throw new AuthenticationError(errorMatch[1], 'AUTH_INVALID_CREDENTIALS');
    }

    // Check for inline error message variable
    const errorMsgMatch = responseText.match(/const errorMessage = '([^']+)'/);
    if (errorMsgMatch?.[1]) {
      throw new AuthenticationError(errorMsgMatch[1], 'AUTH_INVALID_CREDENTIALS');
    }

    // If still on login page with isLoggedIn = false, login failed
    if (responseText.includes('isLoggedIn = false')) {
      throw new AuthenticationError(
        '아이디 또는 비밀번호가 일치하지 않습니다.',
        'AUTH_INVALID_CREDENTIALS'
      );
    }

    // Unexpected response
    throw new AuthenticationError('Unexpected login response', 'AUTH_UNEXPECTED_RESPONSE');
  } catch (error) {
    throw wrapAuthError(error, 'Login');
  }
}
