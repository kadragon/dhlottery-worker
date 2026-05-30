package notify

import (
	"testing"

	"github.com/kadragon/dhlottery-worker/internal/testutil"
)

func TestNotifyBuffersWhenCollectorPresent(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(okResp()))
	col := &Collector{}

	Notify(Payload{Type: Success, Title: "T", Message: "M"}, col)

	if len(col.Payloads()) != 1 {
		t.Errorf("payload should be buffered, got %d", len(col.Payloads()))
	}
	if len(stub.Requests) != 0 {
		t.Errorf("should not send when collecting, got %d requests", len(stub.Requests))
	}
}

func TestNotifySendsWhenNoCollector(t *testing.T) {
	stub := installDoer(t, testutil.Sequence(okResp()))

	Notify(Payload{Type: Success, Title: "T", Message: "M"}, nil)

	if len(stub.Requests) != 1 {
		t.Errorf("should send immediately when no collector, got %d requests", len(stub.Requests))
	}
}
