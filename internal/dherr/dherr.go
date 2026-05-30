// Package dherr defines the error types used across the DHLottery worker.
//
// A single concrete type (Error) carries a stable Code and a Kind that
// distinguishes base, authentication, and purchase errors — mirroring the
// DHLotteryError / AuthenticationError / PurchaseError hierarchy of the
// original TypeScript implementation.
package dherr

// Kind classifies an Error so callers can react to a family of failures
// without matching on individual codes.
type Kind string

const (
	// KindBase is a generic DHLottery error.
	KindBase Kind = "dhlottery"
	// KindAuth is an authentication error.
	KindAuth Kind = "auth"
	// KindPurchase is a purchase error.
	KindPurchase Kind = "purchase"
)

// Error is the common error type for DHLottery operations.
type Error struct {
	Message string
	Code    string
	Kind    Kind
}

// Error implements the error interface.
func (e *Error) Error() string { return e.Message }

// New returns a base DHLottery error.
func New(message, code string) *Error {
	return &Error{Message: message, Code: code, Kind: KindBase}
}

// NewAuth returns an authentication error.
func NewAuth(message, code string) *Error {
	if code == "" {
		code = "AUTH_ERROR"
	}
	return &Error{Message: message, Code: code, Kind: KindAuth}
}

// NewPurchase returns a purchase error.
func NewPurchase(message, code string) *Error {
	if code == "" {
		code = "PURCHASE_ERROR"
	}
	return &Error{Message: message, Code: code, Kind: KindPurchase}
}

// Code extracts the Code from an error if it is a *Error, otherwise "".
func Code(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return ""
}

// WrapAuth wraps a non-authentication error as an authentication error.
// An error that is already an auth *Error is returned unchanged.
func WrapAuth(err error, context string) *Error {
	if e, ok := err.(*Error); ok && e.Kind == KindAuth {
		return e
	}
	msg := "Unknown error"
	if err != nil {
		msg = err.Error()
	}
	return NewAuth(context+" failed: "+msg, "AUTH_NETWORK_ERROR")
}
