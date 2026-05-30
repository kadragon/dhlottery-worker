// Package env is the sole boundary for reading required environment variables.
// Domain code must never read os.Getenv directly.
package env

import (
	"fmt"
	"os"
	"strings"
)

// Required lists the environment variables that must be present at runtime.
var Required = []string{
	"USER_ID",
	"PASSWORD",
	"TELEGRAM_BOT_TOKEN",
	"TELEGRAM_CHAT_ID",
}

// Get returns the value of a required environment variable, or an error if it
// is unset or empty.
func Get(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return v, nil
}

// Validate ensures all required environment variables are present.
func Validate() error {
	var missing []string
	for _, key := range Required {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}
