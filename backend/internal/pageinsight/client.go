package pageinsight

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Fetcher defines how the client retrieves raw HTML.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (body io.ReadCloser, statusCode int, err error)
}

// limitedReadCloser reads from a LimitReader but closes the original body.
type limitedReadCloser struct {
	io.Reader
	io.Closer
}

// HTTPClient implements Fetcher using a real HTTP client.
type HTTPClient struct {
	client *http.Client
}

const (
	maxRedirects = 5
	userAgent    = "PageInsightBot/1.0"
)

var (
	errTooManyRedirects = errors.New("too many redirects")
	errBlockedRedirect  = errors.New("redirect to non-http(s) scheme blocked")
)

// NewHTTPClient returns a Fetcher backed by an http.Client with a 10s timeout,
// a dedicated transport that blocks connections to private/reserved IP ranges,
// and redirect validation that prevents SSRF via redirect chains.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext:         safeDialer().DialContext,
				MaxConnsPerHost:     10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			CheckRedirect: safeRedirectPolicy,
		},
	}
}

// safeRedirectPolicy validates redirect targets and limits the redirect chain length.
func safeRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("%w: stopped after %d", errTooManyRedirects, maxRedirects)
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("%w: %s", errBlockedRedirect, req.URL.Scheme)
	}
	return nil
}

// Fetch retrieves the page at the given URL and returns its body.
func (c *HTTPClient) Fetch(ctx context.Context, targetURL string) (io.ReadCloser, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := c.client.Do(req) //nolint:bodyclose // body is returned to caller via limitedReadCloser
	if err != nil {
		return nil, 0, err
	}

	// Limit response body to 10 MB to prevent memory exhaustion from
	// extremely large or infinite responses.
	const maxResponseBody = 10 << 20
	limited := &limitedReadCloser{
		Reader: io.LimitReader(resp.Body, maxResponseBody),
		Closer: resp.Body,
	}

	return limited, resp.StatusCode, nil
}
