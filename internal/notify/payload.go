// Package notify formats and delivers notifications, optionally batching them
// through a Collector for a single combined send at the end of a workflow.
package notify

// Type is a notification severity.
type Type string

const (
	// Success indicates a successful operation.
	Success Type = "success"
	// Warning indicates a non-fatal warning.
	Warning Type = "warning"
	// Error indicates a failure.
	Error Type = "error"
)

// KV is an ordered detail entry. Insertion order is preserved in the rendered
// message.
type KV struct {
	Key   string
	Value string
}

// Payload is a single notification.
type Payload struct {
	Type    Type
	Title   string
	Message string
	Details []KV
}
