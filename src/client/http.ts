/**
 * HTTP Client with Cookie Management
 *
 * Trace:
 *   spec_id: SPEC-SESSION-001
 *   task_id: TASK-001
 */

import type {
  CookieStore,
  HttpClient,
  HttpRequestOptions,
  HttpResponse,
} from '../types/http.types';

/**
 * Parse Set-Cookie header value to extract cookie name and value
 * Format: "name=value; Path=/; HttpOnly; ..."
 */
function parseCookie(setCookieValue: string): { name: string; value: string } {
  const parts = setCookieValue.split(';');
  const cookiePair = parts[0].trim();
  const [name, value] = cookiePair.split('=');
  return { name: name.trim(), value: value?.trim() || '' };
}

/**
 * Serialize cookie store to Cookie header format
 * Format: "name1=value1; name2=value2"
 */
function serializeCookies(cookies: CookieStore): string {
  return Object.entries(cookies)
    .map(([name, value]) => `${name}=${value}`)
    .join('; ');
}

/**
 * Create HTTP client with automatic cookie management
 */
export function createHttpClient(): HttpClient {
  const cookieStore: CookieStore = {};

  return {
    get cookies(): CookieStore {
      return cookieStore;
    },

    async fetch(url: string, options: HttpRequestOptions = {}): Promise<HttpResponse> {
      // Prepare request headers
      const headers: Record<string, string> = {
        ...options.headers,
      };

      // Include cookies if any are stored
      const cookieHeader = this.getCookieHeader();
      if (cookieHeader) {
        headers.Cookie = cookieHeader;
      }

      // Make request
      const response = await fetch(url, {
        ...options,
        headers,
      });

      // Capture cookies from Set-Cookie headers
      const setCookieHeaders = response.headers.getSetCookie();
      for (const setCookie of setCookieHeaders) {
        const { name, value } = parseCookie(setCookie);
        cookieStore[name] = value;
      }

      // Return response wrapper
      return {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers,
        text: () => response.text(),
        json: <T>() => response.json() as Promise<T>,
      };
    },

    getCookieHeader(): string {
      return serializeCookies(cookieStore);
    },

    clearCookies(): void {
      for (const key of Object.keys(cookieStore)) {
        delete cookieStore[key];
      }
    },
  };
}
