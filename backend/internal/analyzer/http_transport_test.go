package analyzer

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Bahjat/page-insight-tool/backend/internal/model"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/errs"
)

// mockProvider implements PageInsightProvider for testing.
type mockProvider struct {
	result *model.PageAnalysis
	err    error
}

func (m *mockProvider) Analyze(_ context.Context, _ string) (*model.PageAnalysis, error) {
	return m.result, m.err
}

func newTestMux(provider PageInsightProvider) *http.ServeMux {
	logger := slog.Default()
	svc := NewService(provider, logger)
	transport := NewTransport(svc, logger)
	mux := http.NewServeMux()
	transport.RegisterRoutes(mux)
	return mux
}

func TestHandleAnalyze_Success(t *testing.T) {
	provider := &mockProvider{
		result: &model.PageAnalysis{
			URL:         "https://example.com",
			HTMLVersion: "HTML5",
			Title:       "Example",
			Headings:    map[string]int{"h1": 1, "h2": 0, "h3": 0, "h4": 0, "h5": 0, "h6": 0},
			Links:       model.LinkStats{Internal: 3, External: 1, Inaccessible: 0},
		},
	}
	mux := newTestMux(provider)

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result model.PageAnalysis
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Title != "Example" {
		t.Errorf("Title = %q, want %q", result.Title, "Example")
	}
}

func TestHandleAnalyze_EmptyURL(t *testing.T) {
	mux := newTestMux(&mockProvider{})

	body := `{"url": ""}`
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleAnalyze_MissingBody(t *testing.T) {
	mux := newTestMux(&mockProvider{})

	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleAnalyze_InvalidInputError(t *testing.T) {
	provider := &mockProvider{
		err: &errs.AppError{
			Kind:    errs.InvalidInput,
			Message: "bad url",
		},
	}
	mux := newTestMux(provider)

	body := `{"url": "ftp://bad"}`
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleAnalyze_UnreachableError(t *testing.T) {
	provider := &mockProvider{
		err: &errs.AppError{
			Kind:    errs.Unreachable,
			Message: "cannot reach",
			Cause:   context.DeadlineExceeded,
		},
	}
	mux := newTestMux(provider)

	body := `{"url": "https://down.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestHandleAnalyze_Timeout(t *testing.T) {
	provider := &mockProvider{
		err: &errs.AppError{
			Kind:    errs.Timeout,
			Message: "Analysis timed out. The target URL may be slow to respond.",
			Cause:   context.DeadlineExceeded,
		},
	}
	mux := newTestMux(provider)

	body := `{"url": "https://slow.example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}
}

func TestHandleAnalyze_MalformedJSON(t *testing.T) {
	mux := newTestMux(&mockProvider{})

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleAnalyze_WrongMethod(t *testing.T) {
	mux := newTestMux(&mockProvider{})

	req := httptest.NewRequest(http.MethodGet, "/analyze", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// ServeMux returns 405 for method mismatch.
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
