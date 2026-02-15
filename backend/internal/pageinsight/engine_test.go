package pageinsight

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Bahjat/page-insight-tool/backend/internal/platform/errs"
)

var errConnectionRefused = errors.New("connection refused")

// mockFetcher implements Fetcher for testing.
type mockFetcher struct {
	body       string
	statusCode int
	err        error
}

func (m *mockFetcher) Fetch(_ context.Context, _ string) (io.ReadCloser, int, error) {
	if m.err != nil {
		return nil, m.statusCode, m.err
	}
	return io.NopCloser(strings.NewReader(m.body)), m.statusCode, nil
}

// mockLinkChecker implements linkChecker for testing.
type mockLinkChecker struct {
	inaccessible int
}

func (m *mockLinkChecker) CheckLinksWithWorkerPool(_ context.Context, _ []string) int {
	return m.inaccessible
}

func TestEngine_Analyze_Success(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>Test Page</title></head><body>
	<h1>Hello</h1>
	<h2>Sub</h2>
	</body></html>`

	engine := NewEngine(&mockFetcher{body: html, statusCode: 200}, &mockLinkChecker{})

	result, err := engine.Analyze(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Page")
	}
	if result.HTMLVersion != "HTML5" {
		t.Errorf("HTMLVersion = %q, want %q", result.HTMLVersion, "HTML5")
	}
	if result.Headings["h1"] != 1 {
		t.Errorf("h1 = %d, want 1", result.Headings["h1"])
	}
	if result.Headings["h2"] != 1 {
		t.Errorf("h2 = %d, want 1", result.Headings["h2"])
	}
	if result.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", result.URL, "https://example.com")
	}
}

func TestEngine_Analyze_FetchError(t *testing.T) {
	engine := NewEngine(&mockFetcher{err: errConnectionRefused, statusCode: 0}, &mockLinkChecker{})

	_, err := engine.Analyze(context.Background(), "https://down.example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *errs.AppError, got %T", err)
	}
	if appErr.Kind != errs.Unreachable {
		t.Errorf("Kind = %d, want %d (Unreachable)", appErr.Kind, errs.Unreachable)
	}
}

func TestEngine_Analyze_InvalidURL(t *testing.T) {
	engine := NewEngine(&mockFetcher{}, &mockLinkChecker{})

	_, err := engine.Analyze(context.Background(), "not-a-valid-url")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *errs.AppError, got %T", err)
	}
	if appErr.Kind != errs.InvalidInput {
		t.Errorf("Kind = %d, want %d (InvalidInput)", appErr.Kind, errs.InvalidInput)
	}
}
