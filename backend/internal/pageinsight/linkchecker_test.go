package pageinsight

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// testLinkChecker returns a LinkChecker with a default transport (no SSRF
// blocking) so tests can reach httptest servers on localhost.
func testLinkChecker(concurrency int) *LinkChecker {
	return newLinkChecker(concurrency, &http.Transport{
		MaxConnsPerHost:     concurrency,
		MaxIdleConnsPerHost: concurrency,
		IdleConnTimeout:     90 * time.Second,
	})
}

func TestCheckLinks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/ok")
		w.WriteHeader(http.StatusMovedPermanently)
	})
	mux.HandleFunc("/not-found", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/server-error", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/forbidden", func(w http.ResponseWriter, r *http.Request) {
		// Simulate servers that block HEAD but allow GET.
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/method-not-allowed", func(w http.ResponseWriter, r *http.Request) {
		// Simulate servers that return 405 on HEAD but allow GET.
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/true-forbidden", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/unauthorized", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	tests := []struct {
		name     string
		links    []string
		expected int
	}{
		{
			name:     "all accessible",
			links:    []string{ts.URL + "/ok", ts.URL + "/redirect"},
			expected: 0,
		},
		{
			name:     "some inaccessible",
			links:    []string{ts.URL + "/ok", ts.URL + "/not-found", ts.URL + "/server-error"},
			expected: 2,
		},
		{
			name:     "all inaccessible",
			links:    []string{ts.URL + "/not-found", ts.URL + "/server-error"},
			expected: 2,
		},
		{
			name:     "empty list",
			links:    []string{},
			expected: 0,
		},
		{
			name:     "malformed URL counted as inaccessible",
			links:    []string{"://bad-url", ts.URL + "/ok"},
			expected: 1,
		},
		{
			name:     "403 on HEAD triggers GET fallback and succeeds",
			links:    []string{ts.URL + "/forbidden"},
			expected: 0,
		},
		{
			name:     "405 on HEAD triggers GET fallback and succeeds",
			links:    []string{ts.URL + "/method-not-allowed"},
			expected: 0,
		},
		{
			name:     "true 403 on both HEAD and GET is inaccessible",
			links:    []string{ts.URL + "/true-forbidden"},
			expected: 1,
		},
		{
			name:     "401 counted as inaccessible",
			links:    []string{ts.URL + "/unauthorized"},
			expected: 1,
		},
		{
			name:     "true 403 and 404 both counted as inaccessible",
			links:    []string{ts.URL + "/true-forbidden", ts.URL + "/not-found"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := testLinkChecker(10).CheckLinks(context.Background(), tt.links)
			if count != tt.expected {
				t.Errorf("inaccessible = %d, want %d", count, tt.expected)
			}
		})
	}
}

func TestCheckLinks_MaxLinksLimit(t *testing.T) {
	var called int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	links := make([]string, 1100)
	for i := range links {
		links[i] = fmt.Sprintf("%s/page/%d", ts.URL, i)
	}

	testLinkChecker(10).CheckLinks(context.Background(), links)

	if atomic.LoadInt64(&called) > 1000 {
		t.Errorf("checked %d links, should cap at 1000", called)
	}
}

func TestCheckLinks_RespectsContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	links := []string{ts.URL + "/ok", ts.URL + "/ok"}

	_ = testLinkChecker(10).CheckLinks(ctx, links)
}

func TestCheckLinks_BlocksPrivateIPs(t *testing.T) {
	// Verify that the production NewLinkChecker blocks localhost.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Use the real constructor which includes the safe dialer.
	lc := NewLinkChecker(10)
	count := lc.CheckLinks(context.Background(), []string{ts.URL + "/ok"})

	// The request to localhost should fail (blocked by safe dialer),
	// which makes the link appear inaccessible.
	if count != 1 {
		t.Errorf("expected localhost to be blocked (inaccessible=1), got %d", count)
	}
}

func TestCheckLink_ContextCancelledDuringRequest(t *testing.T) {
	// When context is cancelled and client.Do fails, checkLink should return
	// false (not count as inaccessible) because the failure was due to cancellation.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	lc := testLinkChecker(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := lc.checkLink(ctx, ts.URL+"/ok")
	if result {
		t.Error("expected false (not inaccessible) when context is cancelled")
	}
}

func TestGetProbe_GETFallbackFails(t *testing.T) {
	// Server returns 403 on HEAD and 500 on GET: should be inaccessible.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	lc := testLinkChecker(1)
	count := lc.CheckLinks(context.Background(), []string{ts.URL + "/page"})
	if count != 1 {
		t.Errorf("inaccessible = %d, want 1", count)
	}
}

func TestGetProbe_ContextCancelled(t *testing.T) {
	// Server returns 405 on HEAD, triggering getProbe. But context is
	// cancelled before the GET, so it should not count as inaccessible.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	lc := testLinkChecker(1)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after HEAD but before GET by using a single link check directly.
	// We need to test getProbe with cancelled context.
	cancel()
	result := lc.getProbe(ctx, ts.URL+"/page")
	if result {
		t.Error("expected false (not inaccessible) when context is cancelled during getProbe")
	}
}

// BenchmarkCheckLinksLatency benchmarks the worker pool with simulated
// network latency (50ms per request).
func BenchmarkCheckLinksLatency(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	for _, n := range []int{1, 10, 50} {
		links := make([]string, n)
		for i := range links {
			links[i] = ts.URL + "/ok"
		}

		b.Run(fmt.Sprintf("worker_pool_%d", n), func(b *testing.B) {
			lc := testLinkChecker(10)
			b.ResetTimer()
			for range b.N {
				lc.CheckLinks(context.Background(), links)
			}
		})
	}
}
