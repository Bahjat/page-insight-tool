package analyzer

import (
	"context"

	"github.com/Bahjat/page-insight-tool/backend/internal/model"
)

// PageInsightProvider defines the contract for any analysis engine.
type PageInsightProvider interface {
	Analyze(ctx context.Context, targetURL string) (*model.PageAnalysis, error)
}
