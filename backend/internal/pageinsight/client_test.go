package pageinsight

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewHTTPClient(t *testing.T) {
	c := NewHTTPClient()
	if c == nil {
		t.Fatal("NewHTTPClient returned nil")
	}
	if c.client == nil {
		t.Fatal("internal http.Client is nil")
	}
}

func TestHTTPClient_Fetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), userAgent)
		}
		if r.Header.Get("Accept") != "text/html" {
			t.Errorf("Accept = %q, want %q", r.Header.Get("Accept"), "text/html")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "<html><body>Hello</body></html>")
	}))
	defer ts.Close()

	c := &HTTPClient{client: ts.Client()}
	body, status, err := c.Fetch(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = body.Close() }()

	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(data) != "<html><body>Hello</body></html>" {
		t.Errorf("body = %q, want %q", string(data), "<html><body>Hello</body></html>")
	}
}

func TestHTTPClient_Fetch_InvalidURL(t *testing.T) {
	c := NewHTTPClient()
	_, _, err := c.Fetch(context.Background(), "://bad-url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestHTTPClient_Fetch_CancelledContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := &HTTPClient{client: ts.Client()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := c.Fetch(ctx, ts.URL)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestSafeRedirectPolicy(t *testing.T) {
	tests := []struct {
		name    string
		scheme  string
		via     int
		wantErr bool
	}{
		{name: "http within limit", scheme: "https", via: 3, wantErr: false},
		{name: "too many redirects", scheme: "https", via: 5, wantErr: true},
		{name: "blocked ftp scheme", scheme: "ftp", via: 0, wantErr: true},
		{name: "blocked file scheme", scheme: "file", via: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{URL: &url.URL{Scheme: tt.scheme, Host: "example.com"}} //nolint:exhaustruct
			via := make([]*http.Request, tt.via)

			err := safeRedirectPolicy(req, via)
			if (err != nil) != tt.wantErr {
				t.Errorf("safeRedirectPolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
