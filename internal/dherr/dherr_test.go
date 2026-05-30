package dherr

import (
	"errors"
	"testing"
)

func TestConstructors(t *testing.T) {
	base := New("boom", "X")
	if base.Kind != KindBase || base.Code != "X" || base.Error() != "boom" {
		t.Errorf("New = %+v", base)
	}
	auth := NewAuth("nope", "AUTH_X")
	if auth.Kind != KindAuth || auth.Code != "AUTH_X" {
		t.Errorf("NewAuth = %+v", auth)
	}
	if NewAuth("d", "").Code != "AUTH_ERROR" {
		t.Error("NewAuth empty code should default to AUTH_ERROR")
	}
	if NewPurchase("d", "").Code != "PURCHASE_ERROR" {
		t.Error("NewPurchase empty code should default to PURCHASE_ERROR")
	}
}

func TestCode(t *testing.T) {
	if Code(New("m", "C1")) != "C1" {
		t.Error("Code should extract C1")
	}
	if Code(errors.New("plain")) != "" {
		t.Error("Code of non-dherr should be empty")
	}
}

func TestWrapAuth(t *testing.T) {
	authErr := NewAuth("orig", "AUTH_SESSION_INIT_ERROR")
	if got := WrapAuth(authErr, "Session"); got != authErr {
		t.Error("WrapAuth should return an existing auth error unchanged")
	}

	wrapped := WrapAuth(errors.New("network down"), "Login")
	if wrapped.Kind != KindAuth || wrapped.Code != "AUTH_NETWORK_ERROR" {
		t.Errorf("WrapAuth = %+v", wrapped)
	}
	if wrapped.Error() != "Login failed: network down" {
		t.Errorf("WrapAuth message = %q", wrapped.Error())
	}

	// A base (non-auth) dherr should be wrapped, not returned as-is.
	baseErr := New("base", "BASE_X")
	if got := WrapAuth(baseErr, "Login"); got == baseErr || got.Code != "AUTH_NETWORK_ERROR" {
		t.Errorf("WrapAuth(base) = %+v", got)
	}
}
