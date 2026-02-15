package pageinsight

import (
	"context"
	"net/url"

	"github.com/Bahjat/page-insight-tool/backend/internal/model"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/errs"
)

// linkChecker defines how the engine validates link accessibility.
type linkChecker interface {
	CheckLinksWithWorkerPool(ctx context.Context, links []string) int
}

// Engine orchestrates page fetching, HTML parsing, and link checking.
type Engine struct {
	fetcher     Fetcher
	linkChecker linkChecker
}

// NewEngine returns an Engine backed by the given Fetcher and link checker.
func NewEngine(fetcher Fetcher, lc linkChecker) *Engine {
	return &Engine{
		fetcher:     fetcher,
		linkChecker: lc,
	}
}

// Analyze fetches a URL, parses the HTML, and checks links.
func (e *Engine) Analyze(ctx context.Context, targetURL string) (*model.PageAnalysis, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, &errs.AppError{
			Kind:    errs.InvalidInput,
			Message: "Invalid URL format. Please ensure you entered a valid URL (e.g., https://example.com).",
			Cause:   err,
		}
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, &errs.AppError{
			Kind:    errs.InvalidInput,
			Message: "Invalid URL format. Please ensure you entered a valid URL (e.g., https://example.com).",
		}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, &errs.AppError{
			Kind:    errs.InvalidInput,
			Message: "Only http and https URLs are supported.",
		}
	}

	body, statusCode, err := e.fetcher.Fetch(ctx, targetURL)
	if err != nil {
		return nil, &errs.AppError{
			Kind:    errs.Unreachable,
			Message: "The provided URL could not be reached. Check the address.",
			Cause:   err,
		}
	}
	defer func() { _ = body.Close() }()

	if statusCode >= 400 {
		return nil, &errs.AppError{
			Kind:           errs.Unreachable,
			UpstreamStatus: statusCode,
			Message:        "The provided URL returned an error status.",
		}
	}

	parseResult, err := Parse(body, parsed)
	if err != nil {
		return nil, &errs.AppError{
			Kind:    errs.ParsingFailed,
			Message: "Failed to parse the HTML content.",
			Cause:   err,
		}
	}

	// Collect link URLs for accessibility checking.
	var linkURLs []string
	var internalCount, externalCount int
	for _, link := range parseResult.Links {
		linkURLs = append(linkURLs, link.URL)
		if link.IsInternal {
			internalCount++
		} else {
			externalCount++
		}
	}

	inaccessible := e.linkChecker.CheckLinksWithWorkerPool(ctx, linkURLs)

	return &model.PageAnalysis{
		URL:         targetURL,
		HTMLVersion: parseResult.HTMLVersion,
		Title:       parseResult.Title,
		Headings:    parseResult.Headings,
		Links: model.LinkStats{
			Internal:     internalCount,
			External:     externalCount,
			Inaccessible: inaccessible,
		},
		HasLoginForm: parseResult.HasLoginForm,
	}, nil
}
