package analyzer

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Bahjat/page-insight-tool/backend/internal/model"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/errs"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/requestid"
)

// Service orchestrates a PageInsightProvider and logs results.
type Service struct {
	provider PageInsightProvider
	logger   *slog.Logger
}

// NewService creates a Service backed by the given provider.
func NewService(provider PageInsightProvider, logger *slog.Logger) *Service {
	return &Service{provider: provider, logger: logger}
}

// Analyze delegates to the provider and logs the outcome.
func (s *Service) Analyze(ctx context.Context, targetURL string) (*model.PageAnalysis, error) {
	logger := s.logger.With("url", targetURL, "request_id", requestid.FromContext(ctx))

	result, err := s.provider.Analyze(ctx, targetURL)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = &errs.AppError{
				Kind:    errs.Timeout,
				Message: "Analysis timed out. The target URL may be slow to respond.",
				Cause:   err,
			}
		}

		attrs := []any{"error", err}
		var appErr *errs.AppError
		if errors.As(err, &appErr) && appErr.UpstreamStatus != 0 {
			attrs = append(attrs, "target_status", appErr.UpstreamStatus)
		}
		logger.Error("analysis failed", attrs...)
		return nil, err
	}

	logger.Info("analysis complete",
		"title", result.Title,
		"html_version", result.HTMLVersion,
		"has_login_form", result.HasLoginForm,
		"internal_links", result.Links.Internal,
		"external_links", result.Links.External,
		"inaccessible_links", result.Links.Inaccessible,
	)
	return result, nil
}
