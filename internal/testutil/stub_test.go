package testutil

import (
	"net/http"
	"strings"
	"testing"
)

func TestStubDoerRecordsAndResponds(t *testing.T) {
	doer := &StubDoer{Handler: Sequence(JSON(`{"ok":true}`), StubResponse{Status: 500, Body: "err"})}

	req1, _ := http.NewRequest(http.MethodPost, "https://example.com/a", strings.NewReader("body1"))
	resp1, err := doer.Do(req1)
	if err != nil {
		t.Fatal(err)
	}
	if resp1.StatusCode != 200 {
		t.Errorf("resp1 status = %d", resp1.StatusCode)
	}
	if len(doer.Requests) != 1 || doer.Requests[0].Body != "body1" || doer.Requests[0].Method != http.MethodPost {
		t.Errorf("recorded = %+v", doer.Requests)
	}

	req2, _ := http.NewRequest(http.MethodGet, "https://example.com/b", nil)
	resp2, _ := doer.Do(req2)
	if resp2.StatusCode != 500 {
		t.Errorf("resp2 status = %d", resp2.StatusCode)
	}

	// Extra calls reuse the last response.
	req3, _ := http.NewRequest(http.MethodGet, "https://example.com/c", nil)
	resp3, _ := doer.Do(req3)
	if resp3.StatusCode != 500 {
		t.Errorf("resp3 status = %d", resp3.StatusCode)
	}
}
