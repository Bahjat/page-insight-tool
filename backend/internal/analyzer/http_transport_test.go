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

func TestHandleAnalyze_ErrorCases(t *testing.T) {
	mux := newTestMux(&mockProvider{})

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{"empty URL", http.MethodPost, `{"url": ""}`, http.StatusBadRequest},
		{"missing body", http.MethodPost, "", http.StatusBadRequest},
		{"malformed JSON", http.MethodPost, `{invalid json`, http.StatusBadRequest},
		{"wrong method", http.MethodGet, "", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" && tt.method == http.MethodGet {
				req = httptest.NewRequest(tt.method, "/analyze", nil)
			} else {
				req = httptest.NewRequest(tt.method, "/analyze", strings.NewReader(tt.body))
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandleAnalyze_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		body       string
		wantStatus int
	}{
		{
			"invalid input",
			&errs.AppError{Kind: errs.InvalidInput, Message: "bad url"},
			`{"url": "ftp://bad"}`,
			http.StatusBadRequest,
		},
		{
			"unreachable",
			&errs.AppError{Kind: errs.Unreachable, Message: "cannot reach", Cause: context.DeadlineExceeded},
			`{"url": "https://down.example.com"}`,
			http.StatusBadGateway,
		},
		{
			"timeout",
			&errs.AppError{Kind: errs.Timeout, Message: "Analysis timed out. The target URL may be slow to respond.", Cause: context.DeadlineExceeded},
			`{"url": "https://slow.example.com"}`,
			http.StatusGatewayTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newTestMux(&mockProvider{err: tt.err})
			req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
