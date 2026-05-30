package notify

import "testing"

func TestCollectorAddAndPayloads(t *testing.T) {
	c := &Collector{}
	if !c.IsEmpty() {
		t.Error("new collector should be empty")
	}
	p := Payload{Type: Success, Title: "Test", Message: "Test message"}
	c.Add(p)
	if c.IsEmpty() {
		t.Error("collector should not be empty after Add")
	}
	got := c.Payloads()
	if len(got) != 1 || got[0].Type != p.Type || got[0].Title != p.Title || got[0].Message != p.Message {
		t.Errorf("Payloads() = %+v", got)
	}
}

func TestCollectorSnapshotIsCopy(t *testing.T) {
	c := &Collector{}
	c.Add(Payload{Type: Success, Title: "Test", Message: "m"})
	snapshot := c.Payloads()
	snapshot = append(snapshot, Payload{Type: Error, Title: "Extra", Message: "x"})
	_ = snapshot
	if len(c.Payloads()) != 1 {
		t.Errorf("mutating snapshot must not affect collector: %d", len(c.Payloads()))
	}
}
