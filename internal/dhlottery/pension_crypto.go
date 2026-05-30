package dhlottery

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"

	"github.com/kadragon/dhlottery-worker/internal/dherr"
)

// EL "q" payload encryption (reverse-engineered to match DHLottery's jsbn.js):
//   - passphrase: first 32 chars of the session id
//   - PBKDF2-SHA256 (iterations: 1000, key size: 128 bit)
//   - AES-128-CBC + PKCS7
//   - payload: salt(32-byte hex) + iv(16-byte hex) + ciphertext(base64),
//     then encodeURIComponent.
const (
	pbkdf2Iterations = 1000
	keyBytes         = 16 // 128-bit
	saltBytes        = 32
	ivBytes          = 16
	saltHexLength    = saltBytes * 2
	ivHexLength      = ivBytes * 2
)

func getPassphrase(sessionID string) (string, error) {
	if len(sessionID) < 32 {
		return "", dherr.New("Session cookie is missing or too short for EL encryption", "PENSION_INVALID_SESSION")
	}
	return sessionID[:32], nil
}

func deriveKey(passphrase string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, passphrase, salt, pbkdf2Iterations, keyBytes)
}

func decodePossiblyEncoded(input string) string {
	// PathUnescape (not QueryUnescape) matches JavaScript's decodeURIComponent:
	// it decodes %XX but leaves a literal '+' as '+'. QueryUnescape would turn
	// '+' into a space, corrupting raw base64 ciphertext returned by the server
	// (the response is not URL-encoded, so '+' characters appear verbatim).
	if decoded, err := url.PathUnescape(input); err == nil {
		return decoded
	}
	return input
}

// EncryptElQ encrypts plainText into the EL "q" payload format.
func EncryptElQ(plainText, sessionID string) (string, error) {
	passphrase, err := getPassphrase(sessionID)
	if err != nil {
		return "", err
	}

	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	iv := make([]byte, ivBytes)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	padded := pkcs7Pad([]byte(plainText), block.BlockSize())
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	combined := hex.EncodeToString(salt) + hex.EncodeToString(iv) + base64.StdEncoding.EncodeToString(ciphertext)
	return url.QueryEscape(combined), nil
}

// DecryptElQ decrypts an EL "q" payload back to plaintext.
func DecryptElQ(encryptedQ, sessionID string) (string, error) {
	passphrase, err := getPassphrase(sessionID)
	if err != nil {
		return "", err
	}

	decoded := decodePossiblyEncoded(encryptedQ)
	if len(decoded) < saltHexLength+ivHexLength+1 {
		return "", dherr.New("Invalid EL encrypted payload format", "PENSION_DECRYPT_FAILED")
	}

	plain, err := decryptElPayload(decoded, passphrase)
	if err != nil {
		return "", dherr.New("Failed to decrypt EL payload: "+err.Error(), "PENSION_DECRYPT_FAILED")
	}
	return plain, nil
}

func decryptElPayload(decoded, passphrase string) (string, error) {
	salt, err := hex.DecodeString(decoded[:saltHexLength])
	if err != nil {
		return "", err
	}
	iv, err := hex.DecodeString(decoded[saltHexLength : saltHexLength+ivHexLength])
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(decoded[saltHexLength+ivHexLength:])
	if err != nil {
		return "", err
	}

	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(ciphertext) == 0 || len(ciphertext)%block.BlockSize() != 0 {
		return "", fmt.Errorf("invalid ciphertext length")
	}

	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, ciphertext)
	return pkcs7Unpad(plain, block.BlockSize())
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - len(data)%blockSize
	pad := make([]byte, padLen)
	for i := range pad {
		pad[i] = byte(padLen) //nolint:gosec // G115: padLen = blockSize - x%blockSize; AES blockSize=16, so 1≤padLen≤16, safe
	}
	return append(data, pad...)
}

func pkcs7Unpad(data []byte, blockSize int) (string, error) {
	n := len(data)
	if n == 0 || n%blockSize != 0 {
		return "", fmt.Errorf("invalid padded data length")
	}
	padLen := int(data[n-1])
	if padLen == 0 || padLen > blockSize {
		return "", fmt.Errorf("invalid padding")
	}
	for _, b := range data[n-padLen:] {
		if int(b) != padLen {
			return "", fmt.Errorf("invalid padding")
		}
	}
	return string(data[:n-padLen]), nil
}
