// Package httpclient is an HTTP client with automatic, ordered cookie
// management and manual redirect handling (3xx responses are returned as-is so
// callers can inspect Location and preserve cookies across hops).
package httpclient

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/logger"
	"golang.org/x/text/encoding/korean"
)

// Doer performs an HTTP request. *http.Client satisfies it; tests inject fakes.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// RequestOptions configures a single request.
type RequestOptions struct {
	Method  string
	Headers map[string]string
	Body    string
}

// Response is a buffered HTTP response with decoding helpers.
type Response struct {
	Status     int
	StatusText string
	Header     http.Header
	body       []byte
}

// Text decodes the body. encoding may be "utf-8" (default) or "euc-kr".
func (r *Response) Text(encoding string) (string, error) {
	if encoding == "euc-kr" {
		decoded, err := korean.EUCKR.NewDecoder().Bytes(r.body)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}
	return string(r.body), nil
}

// JSON unmarshals the UTF-8 body into v.
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.body, v)
}

// Client manages cookies across requests.
type Client struct {
	doer    Doer
	order   []string
	cookies map[string]string
}

// New returns a Client backed by a real http.Client that does not follow
// redirects (so 3xx responses are returned to the caller).
func New() *Client {
	return NewWithDoer(&http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	})
}

// NewWithDoer returns a Client backed by the given Doer. Intended for tests.
func NewWithDoer(d Doer) *Client {
	return &Client{doer: d, cookies: map[string]string{}}
}

// Cookie returns the stored value for name, or "".
func (c *Client) Cookie(name string) string {
	return c.cookies[name]
}

// SetCookie sets a cookie value directly (used to seed sessions in tests).
func (c *Client) SetCookie(name, value string) {
	c.store(name, value)
}

// CookieHeader returns the serialized Cookie header in insertion order.
func (c *Client) CookieHeader() string {
	parts := make([]string, 0, len(c.order))
	for _, name := range c.order {
		parts = append(parts, name+"="+c.cookies[name])
	}
	return strings.Join(parts, "; ")
}

// ClearCookies removes all stored cookies.
func (c *Client) ClearCookies() {
	c.order = nil
	c.cookies = map[string]string{}
}

func (c *Client) store(name, value string) {
	if _, exists := c.cookies[name]; !exists {
		c.order = append(c.order, name)
	}
	c.cookies[name] = value
}

func (c *Client) capture(name, value string) {
	if existing, exists := c.cookies[name]; exists && existing != value {
		logger.Warn("Cookie overwritten with different value", logger.Fields{
			"event": "cookie_overwritten",
			"name":  name,
		})
	}
	c.store(name, value)
}

// Fetch performs a request, injecting stored cookies and capturing Set-Cookie.
func (c *Client) Fetch(rawURL string, opts RequestOptions) (*Response, error) {
	method := opts.Method
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if opts.Body != "" {
		body = strings.NewReader(opts.Body)
	}

	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}
	if cookieHeader := c.CookieHeader(); cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	for _, setCookie := range resp.Header.Values("Set-Cookie") {
		name, value := parseCookie(setCookie)
		if name != "" {
			c.capture(name, value)
		}
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		Header:     resp.Header,
		body:       raw,
	}, nil
}

// parseCookie extracts the name and value from a Set-Cookie header value.
// It splits on the first '=' so values containing '=' (e.g. "abc.def==") are
// preserved.
func parseCookie(setCookie string) (name, value string) {
	pair := setCookie
	if idx := strings.IndexByte(pair, ';'); idx >= 0 {
		pair = pair[:idx]
	}
	pair = strings.TrimSpace(pair)
	eq := strings.IndexByte(pair, '=')
	if eq < 0 {
		return strings.TrimSpace(pair), ""
	}
	return strings.TrimSpace(pair[:eq]), strings.TrimSpace(pair[eq+1:])
}
