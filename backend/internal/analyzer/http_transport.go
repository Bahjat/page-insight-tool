package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Bahjat/page-insight-tool/backend/internal/model"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/errs"
)

const analyzeTimeout = 60 * time.Second

var errURLRequired = errors.New("the \"url\" field is required")

// Transport handles HTTP requests for page analysis.
type Transport struct {
	service *Service
	logger  *slog.Logger
}

// NewTransport creates an HTTP transport backed by the given service.
func NewTransport(service *Service, logger *slog.Logger) *Transport {
	return &Transport{service: service, logger: logger}
}

// RegisterRoutes attaches the transport's handlers to the given mux.
func (t *Transport) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /analyze", t.handleAnalyze)
}

type analyzeRequest struct {
	URL string `json:"url"`
}

func (r analyzeRequest) validate() error {
	if r.URL == "" {
		return errURLRequired
	}
	return nil
}

func (t *Transport) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	const maxRequestBody = 1 << 20 // 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var req analyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.renderError(w, http.StatusBadRequest, "Invalid request body. Please send a JSON object with a \"url\" field.")
		return
	}

	if err := req.validate(); err != nil {
		t.renderError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), analyzeTimeout)
	defer cancel()

	result, err := t.service.Analyze(ctx, req.URL)
	if err != nil {
		t.handleServiceError(w, err)
		return
	}

	t.renderJSON(w, http.StatusOK, result)
}

func (t *Transport) handleServiceError(w http.ResponseWriter, err error) {
	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		status := http.StatusInternalServerError
		switch appErr.Kind {
		case errs.InvalidInput:
			status = http.StatusBadRequest
		case errs.Unreachable:
			status = http.StatusBadGateway
		case errs.Timeout:
			status = http.StatusGatewayTimeout
		case errs.ParsingFailed, errs.Unknown:
			// 500 Internal Server Error
		}
		t.renderError(w, status, appErr.Message)
		return
	}

	t.renderError(w, http.StatusInternalServerError, "An unexpected error occurred.")
}

func (t *Transport) renderJSON(w http.ResponseWriter, status int, data any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		t.logger.Error("failed to encode response", "error", err)
		http.Error(w, `{"error":"Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func (t *Transport) renderError(w http.ResponseWriter, status int, message string) {
	t.renderJSON(w, status, model.ErrorResponse{
		Error:      http.StatusText(status),
		StatusCode: status,
		Message:    message,
	})
}
