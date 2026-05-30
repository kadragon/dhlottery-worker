package env

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	t.Setenv("USER_ID", "alice")
	if v, err := Get("USER_ID"); err != nil || v != "alice" {
		t.Errorf("Get(USER_ID) = %q, %v", v, err)
	}

	t.Setenv("PASSWORD", "")
	if _, err := Get("PASSWORD"); err == nil {
		t.Error("Get should error on empty value")
	} else if !strings.Contains(err.Error(), "PASSWORD") {
		t.Errorf("error should name the key: %v", err)
	}
}

func TestValidate(t *testing.T) {
	t.Setenv("USER_ID", "u")
	t.Setenv("PASSWORD", "p")
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "c")
	if err := Validate(); err != nil {
		t.Errorf("Validate should pass: %v", err)
	}

	t.Setenv("TELEGRAM_CHAT_ID", "")
	err := Validate()
	if err == nil || !strings.Contains(err.Error(), "TELEGRAM_CHAT_ID") {
		t.Errorf("Validate should report missing TELEGRAM_CHAT_ID: %v", err)
	}
}
