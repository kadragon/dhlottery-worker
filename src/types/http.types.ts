/**
 * HTTP Client Types
 *
 * Trace:
 *   spec_id: SPEC-SESSION-001
 *   task_id: TASK-001
 */

/**
 * Cookie storage structure
 * Maps cookie name to cookie value
 */
export interface CookieStore {
  [name: string]: string;
}

/**
 * HTTP request options
 */
export interface HttpRequestOptions {
  method?: string;
  headers?: Record<string, string>;
  body?: string | FormData;
}

/**
 * HTTP response with cookie handling
 */
export interface HttpResponse {
  status: number;
  statusText: string;
  headers: Headers;
  text: (encoding?: 'utf-8' | 'euc-kr') => Promise<string>;
  json: <T>() => Promise<T>;
}

/**
 * HTTP Client interface with cookie management
 */
export interface HttpClient {
  /**
   * Current cookie store
   */
  readonly cookies: CookieStore;

  /**
   * Make HTTP request with automatic cookie handling
   * - Includes stored cookies in request
   * - Captures Set-Cookie from response
   * - Updates cookie store
   */
  fetch(url: string, options?: HttpRequestOptions): Promise<HttpResponse>;

  /**
   * Get serialized cookie string for Cookie header
   */
  getCookieHeader(): string;

  /**
   * Clear all stored cookies
   */
  clearCookies(): void;
}
