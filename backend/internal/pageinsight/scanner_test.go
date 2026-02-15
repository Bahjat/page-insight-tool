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

func TestCheckLinksWithWorkerPool(t *testing.T) {
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
	mux.HandleFunc("/forbidden", func(w http.ResponseWriter, _ *http.Request) {
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
			name:     "403 counted as inaccessible",
			links:    []string{ts.URL + "/forbidden"},
			expected: 1,
		},
		{
			name:     "401 counted as inaccessible",
			links:    []string{ts.URL + "/unauthorized"},
			expected: 1,
		},
		{
			name:     "403 and 404 both counted as inaccessible",
			links:    []string{ts.URL + "/forbidden", ts.URL + "/not-found"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := testLinkChecker(10).CheckLinksWithWorkerPool(context.Background(), tt.links)
			if count != tt.expected {
				t.Errorf("inaccessible = %d, want %d", count, tt.expected)
			}
		})
	}
}

func TestCheckLinksWithWorkerPool_MaxLinksLimit(t *testing.T) {
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

	testLinkChecker(10).CheckLinksWithWorkerPool(context.Background(), links)

	if atomic.LoadInt64(&called) > 1000 {
		t.Errorf("checked %d links, should cap at 1000", called)
	}
}

func TestCheckLinksWithWorkerPool_RespectsContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	links := []string{ts.URL + "/ok", ts.URL + "/ok"}

	_ = testLinkChecker(10).CheckLinksWithWorkerPool(ctx, links)
}

func TestCheckLinksWithWorkerPool_BlocksPrivateIPs(t *testing.T) {
	// Verify that the production NewLinkChecker blocks localhost.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Use the real constructor which includes the safe dialer.
	lc := NewLinkChecker(10)
	count := lc.CheckLinksWithWorkerPool(context.Background(), []string{ts.URL + "/ok"})

	// The request to localhost should fail (blocked by safe dialer),
	// which makes the link appear inaccessible.
	if count != 1 {
		t.Errorf("expected localhost to be blocked (inaccessible=1), got %d", count)
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
				lc.CheckLinksWithWorkerPool(context.Background(), links)
			}
		})
	}
}
