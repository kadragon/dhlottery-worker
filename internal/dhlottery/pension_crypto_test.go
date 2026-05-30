package dhlottery

import (
	"net/url"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	sessionID := "12345678901234567890123456789012.rest.of.session.id=="
	plain := "ROUND=303&reserveJo=0&repeatRoundCnt=1&totalBuyAmt=5000"

	encrypted, err := EncryptElQ(plain, sessionID)
	if err != nil {
		t.Fatalf("EncryptElQ: %v", err)
	}
	decrypted, err := DecryptElQ(encrypted, sessionID)
	if err != nil {
		t.Fatalf("DecryptElQ: %v", err)
	}
	if decrypted != plain {
		t.Errorf("round-trip = %q, want %q", decrypted, plain)
	}
}

// decodePossiblyEncoded must match JS decodeURIComponent: decode %XX but keep a
// literal '+' (QueryUnescape would turn '+' into a space and corrupt base64).
func TestDecodePreservesPlus(t *testing.T) {
	if got := decodePossiblyEncoded("a+b%2Fc"); got != "a+b/c" {
		t.Errorf("decodePossiblyEncoded = %q, want a+b/c", got)
	}
}

// The server returns a non-URL-encoded payload whose base64 may contain literal
// '+' and '/'. Decryption must still succeed.
func TestDecryptAcceptsRawPayload(t *testing.T) {
	session := "12345678901234567890123456789012.tail.of.session=="
	plain := "ROUND=303&reserveJo=0&totalBuyAmt=5000"

	encrypted, err := EncryptElQ(plain, session)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a server response that is NOT URL-encoded (literal +,/,= present).
	raw, err := url.QueryUnescape(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := DecryptElQ(raw, session)
	if err != nil {
		t.Fatalf("DecryptElQ on raw payload: %v", err)
	}
	if decrypted != plain {
		t.Errorf("decrypted = %q, want %q", decrypted, plain)
	}
}

func TestCryptoShortSession(t *testing.T) {
	if _, err := EncryptElQ("test", ""); err == nil || !strings.Contains(err.Error(), "Session cookie is missing") {
		t.Errorf("EncryptElQ empty session err = %v", err)
	}
	if _, err := DecryptElQ("abc", "short-session"); err == nil || !strings.Contains(err.Error(), "Session cookie is missing") {
		t.Errorf("DecryptElQ short session err = %v", err)
	}
}
