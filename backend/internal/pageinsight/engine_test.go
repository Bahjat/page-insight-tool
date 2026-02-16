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
	receivedURLs []string
}

func (m *mockLinkChecker) CheckLinks(_ context.Context, links []string) int {
	m.receivedURLs = links
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

func TestEngine_Analyze_DeduplicatesLinks(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>Dedup</title></head><body>
	<a href="https://example.com/a">A</a>
	<a href="https://other.com/b">B</a>
	<a href="https://example.com/a">A again</a>
	<a href="https://other.com/b">B again</a>
	<a href="https://example.com/c">C</a>
	</body></html>`

	lc := &mockLinkChecker{}
	engine := NewEngine(&mockFetcher{body: html, statusCode: 200}, lc)

	result, err := engine.Analyze(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Counts should reflect all links including duplicates.
	if result.Links.Internal != 3 {
		t.Errorf("internal = %d, want 3", result.Links.Internal)
	}
	if result.Links.External != 2 {
		t.Errorf("external = %d, want 2", result.Links.External)
	}

	// The link checker should receive only unique URLs.
	if len(lc.receivedURLs) != 3 {
		t.Errorf("unique URLs sent to checker = %d, want 3: %v", len(lc.receivedURLs), lc.receivedURLs)
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

func TestEngine_Analyze_NonHTTPScheme(t *testing.T) {
	engine := NewEngine(&mockFetcher{}, &mockLinkChecker{})

	_, err := engine.Analyze(context.Background(), "ftp://example.com/file")
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

func TestEngine_Analyze_HTTPStatusError(t *testing.T) {
	engine := NewEngine(&mockFetcher{body: "not found", statusCode: 404}, &mockLinkChecker{})

	_, err := engine.Analyze(context.Background(), "https://example.com/missing")
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
	if appErr.UpstreamStatus != 404 {
		t.Errorf("UpstreamStatus = %d, want 404", appErr.UpstreamStatus)
	}
}

func TestEngine_Analyze_LoginFormDetected(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>Login</title></head><body>
	<form><input type="password" name="pw"></form>
	</body></html>`

	engine := NewEngine(&mockFetcher{body: html, statusCode: 200}, &mockLinkChecker{})

	result, err := engine.Analyze(context.Background(), "https://example.com/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasLoginForm {
		t.Error("HasLoginForm = false, want true")
	}
}

func TestEngine_Analyze_InaccessibleCount(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>T</title></head><body>
	<a href="https://example.com/a">A</a>
	<a href="https://other.com/b">B</a>
	</body></html>`

	engine := NewEngine(
		&mockFetcher{body: html, statusCode: 200},
		&mockLinkChecker{inaccessible: 1},
	)

	result, err := engine.Analyze(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Links.Inaccessible != 1 {
		t.Errorf("Inaccessible = %d, want 1", result.Links.Inaccessible)
	}
}
