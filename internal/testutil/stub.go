// Package testutil provides a recording, programmable HTTP Doer for tests.
package testutil

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RecordedRequest is a captured outbound request.
type RecordedRequest struct {
	Method string
	URL    string
	Header http.Header
	Body   string
}

// StubResponse is a canned response. Header may be nil.
type StubResponse struct {
	Status int
	Header http.Header
	Body   string
}

// StubDoer records requests and produces responses via Handler. The handler
// receives the zero-based call index and the recorded request.
type StubDoer struct {
	Handler  func(call int, req RecordedRequest) (StubResponse, error)
	Requests []RecordedRequest
}

// Do implements the Doer interface used by httpclient and notify.
func (d *StubDoer) Do(req *http.Request) (*http.Response, error) {
	var body string
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	rec := RecordedRequest{
		Method: req.Method,
		URL:    req.URL.String(),
		Header: req.Header.Clone(),
		Body:   body,
	}
	call := len(d.Requests)
	d.Requests = append(d.Requests, rec)

	resp, err := d.Handler(call, rec)
	if err != nil {
		return nil, err
	}
	header := resp.Header
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: resp.Status,
		Status:     fmt.Sprintf("%d %s", resp.Status, http.StatusText(resp.Status)),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(resp.Body)),
	}, nil
}

// Sequence returns a handler that replies with the given responses in order.
// Extra calls reuse the last response.
func Sequence(responses ...StubResponse) func(int, RecordedRequest) (StubResponse, error) {
	return func(call int, _ RecordedRequest) (StubResponse, error) {
		if call >= len(responses) {
			return responses[len(responses)-1], nil
		}
		return responses[call], nil
	}
}

// JSON builds a 200 JSON StubResponse.
func JSON(body string) StubResponse {
	return StubResponse{Status: 200, Body: body}
}
