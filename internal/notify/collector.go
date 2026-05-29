package notify

// Collector batches notification payloads for a single combined send.
type Collector struct {
	payloads []Payload
}

// Add appends a payload.
func (c *Collector) Add(payload Payload) {
	c.payloads = append(c.payloads, payload)
}

// Payloads returns a snapshot copy of the collected payloads.
func (c *Collector) Payloads() []Payload {
	out := make([]Payload, len(c.payloads))
	copy(out, c.payloads)
	return out
}

// IsEmpty reports whether no payloads have been collected.
func (c *Collector) IsEmpty() bool {
	return len(c.payloads) == 0
}
