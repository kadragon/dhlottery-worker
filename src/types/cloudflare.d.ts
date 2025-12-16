/**
 * Cloudflare Workers Type Extensions
 *
 * Extends standard Web API types with Cloudflare Workers-specific methods
 */

interface Headers {
  /**
   * Returns an array of all Set-Cookie header values
   * Cloudflare Workers-specific method
   * @see https://developers.cloudflare.com/workers/runtime-apis/headers/#methods
   */
  getSetCookie(): string[];
}
