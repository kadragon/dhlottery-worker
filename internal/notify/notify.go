package notify

// Notify routes a payload: when a collector is provided it is buffered for a
// later combined send, otherwise it is sent immediately.
func Notify(payload Payload, collector *Collector) {
	if collector != nil {
		collector.Add(payload)
		return
	}
	SendNotification(payload)
}
