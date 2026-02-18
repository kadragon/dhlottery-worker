/**
 * Pension 720+ EL encryption utilities
 *
 * EL q payload:
 * - passphrase: first 32 chars of JSESSIONID
 * - PBKDF2-SHA256 (iterations: 1000, key size: 128 bit)
 * - AES-128-CBC + PKCS7
 * - payload: salt(32-byte hex) + iv(16-byte hex) + ciphertext(base64)
 */

import crypto from 'node:crypto';
import { DHLotteryError } from '../utils/errors';

// Low by modern standards, but reverse-engineered to match DHLottery's jsbn.js implementation exactly.
const PBKDF2_ITERATIONS = 1000;
const KEY_BYTES = 16; // 128-bit
const SALT_BYTES = 32;
const IV_BYTES = 16;
const SALT_HEX_LENGTH = SALT_BYTES * 2;
const IV_HEX_LENGTH = IV_BYTES * 2;

function getPassphrase(sessionId: string): string {
  if (!sessionId || sessionId.length < 32) {
    throw new DHLotteryError(
      'Session cookie is missing or too short for EL encryption',
      'PENSION_INVALID_SESSION'
    );
  }
  return sessionId.slice(0, 32);
}

function deriveKey(passphrase: string, salt: Buffer): Buffer {
  return crypto.pbkdf2Sync(passphrase, salt, PBKDF2_ITERATIONS, KEY_BYTES, 'sha256');
}

function decodePossiblyEncoded(input: string): string {
  // encryptElQ URL-encodes its output once; the API response is JSON which
  // does not add further encoding, so a single decode is always sufficient.
  try {
    return decodeURIComponent(input);
  } catch {
    return input;
  }
}

export function encryptElQ(plainText: string, sessionId: string): string {
  const passphrase = getPassphrase(sessionId);
  const salt = crypto.randomBytes(SALT_BYTES);
  const iv = crypto.randomBytes(IV_BYTES);
  const key = deriveKey(passphrase, salt);

  const cipher = crypto.createCipheriv('aes-128-cbc', key, iv);
  const encrypted = Buffer.concat([cipher.update(plainText, 'utf8'), cipher.final()]);

  const combined = `${salt.toString('hex')}${iv.toString('hex')}${encrypted.toString('base64')}`;
  return encodeURIComponent(combined);
}

export function decryptElQ(encryptedQ: string, sessionId: string): string {
  const passphrase = getPassphrase(sessionId);
  const decodedQ = decodePossiblyEncoded(encryptedQ);
  const minimumLength = SALT_HEX_LENGTH + IV_HEX_LENGTH + 1;

  if (decodedQ.length < minimumLength) {
    throw new DHLotteryError('Invalid EL encrypted payload format', 'PENSION_DECRYPT_FAILED');
  }

  try {
    const saltHex = decodedQ.slice(0, SALT_HEX_LENGTH);
    const ivHex = decodedQ.slice(SALT_HEX_LENGTH, SALT_HEX_LENGTH + IV_HEX_LENGTH);
    const ciphertextBase64 = decodedQ.slice(SALT_HEX_LENGTH + IV_HEX_LENGTH);

    const salt = Buffer.from(saltHex, 'hex');
    const iv = Buffer.from(ivHex, 'hex');
    const key = deriveKey(passphrase, salt);

    const decipher = crypto.createDecipheriv('aes-128-cbc', key, iv);
    const decrypted = Buffer.concat([
      decipher.update(ciphertextBase64, 'base64'),
      decipher.final(),
    ]);
    return decrypted.toString('utf8');
  } catch (error) {
    throw new DHLotteryError(
      `Failed to decrypt EL payload: ${error instanceof Error ? error.message : String(error)}`,
      'PENSION_DECRYPT_FAILED'
    );
  }
}
