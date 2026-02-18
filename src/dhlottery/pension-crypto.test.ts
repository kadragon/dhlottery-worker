import { describe, expect, it } from 'vitest';
import { decryptElQ, encryptElQ } from './pension-crypto';

describe('Pension EL crypto', () => {
  const sessionId = '12345678901234567890123456789012.rest.of.session.id==';

  it('should encrypt and decrypt q payload round-trip', () => {
    const plain = 'ROUND=303&reserveJo=0&repeatRoundCnt=1&totalBuyAmt=5000';
    const encrypted = encryptElQ(plain, sessionId);
    const decrypted = decryptElQ(encrypted, sessionId);

    expect(decrypted).toBe(plain);
  });

  it('should decrypt payload that was URL-encoded twice', () => {
    const plain = 'q=test+value&arr=A%2CB';
    const encrypted = encryptElQ(plain, sessionId);
    const doubleEncoded = encodeURIComponent(encrypted);
    const decrypted = decryptElQ(doubleEncoded, sessionId);

    expect(decrypted).toBe(plain);
  });

  it('should throw when session id is missing or too short', () => {
    expect(() => encryptElQ('test', '')).toThrow('Session cookie is missing');
    expect(() => decryptElQ('abc', 'short-session')).toThrow('Session cookie is missing');
  });
});
