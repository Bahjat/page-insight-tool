package pageinsight

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const maxLinks = 1000

// LinkChecker validates link accessibility using a reusable HTTP client.
type LinkChecker struct {
	client      *http.Client
	concurrency int
}

// NewLinkChecker returns a LinkChecker with a 5s timeout that does not follow
// redirects and blocks connections to private/reserved IP ranges.
// The concurrency parameter controls the worker pool size.
func NewLinkChecker(concurrency int) *LinkChecker {
	return newLinkChecker(concurrency, &http.Transport{
		DialContext:         safeDialer().DialContext,
		MaxConnsPerHost:     concurrency,
		MaxIdleConnsPerHost: concurrency,
		IdleConnTimeout:     90 * time.Second,
	})
}

func newLinkChecker(concurrency int, transport http.RoundTripper) *LinkChecker {
	return &LinkChecker{
		concurrency: concurrency,
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// checkLink performs a HEAD request and returns true if the link is inaccessible.
func (lc *LinkChecker) checkLink(ctx context.Context, link string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
	if err != nil {
		return true // malformed URL is inaccessible
	}

	resp, err := lc.client.Do(req)
	if err != nil {
		return ctx.Err() == nil // inaccessible only if context wasn't cancelled
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode >= 400
}

// CheckLinksWithWorkerPool validates a list of URLs concurrently using a pool
// of worker goroutines sized by the configured concurrency and returns the
// count of inaccessible links. Processes at most 1000 links.
func (lc *LinkChecker) CheckLinksWithWorkerPool(ctx context.Context, links []string) int {
	limit := min(len(links), maxLinks)
	links = links[:limit]

	if limit == 0 {
		return 0
	}

	jobs := make(chan string, limit)
	results := make(chan bool, limit)

	numWorkers := min(limit, lc.concurrency)

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			for link := range jobs {
				results <- lc.checkLink(ctx, link)
			}
		})
	}

	for _, link := range links {
		jobs <- link
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var inaccessible int
	for bad := range results {
		if bad {
			inaccessible++
		}
	}

	return inaccessible
}
