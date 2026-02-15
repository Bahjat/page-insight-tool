package pageinsight

import (
	"errors"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ParseResult holds everything extracted from a single-pass HTML parse.
type ParseResult struct {
	HTMLVersion  string
	Title        string
	Headings     map[string]int
	Links        []Link
	HasLoginForm bool
}

// Link represents a URL found on the page with its classification.
type Link struct {
	URL        string
	IsInternal bool
}

// Parse performs a single-pass traversal of the HTML body, extracting
// title, headings, HTML version, links, and login form presence.
func Parse(body io.Reader, baseURL *url.URL) (*ParseResult, error) {
	result := &ParseResult{
		HTMLVersion: "Unknown",
		Headings:    map[string]int{"h1": 0, "h2": 0, "h3": 0, "h4": 0, "h5": 0, "h6": 0},
	}

	z := html.NewTokenizer(body)
	var inTitle bool

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if errors.Is(z.Err(), io.EOF) {
				return result, nil
			}
			return nil, z.Err()

		case html.DoctypeToken:
			token := z.Token()
			result.HTMLVersion = detectHTMLVersion(token)

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := z.TagName()
			tag := string(tn)

			switch {
			case tag == "title":
				inTitle = true

			case isHeading(tag):
				result.Headings[tag]++

			case tag == "a" && hasAttr:
				if href := extractAttr(z, "href"); href != "" {
					if link, ok := classifyLink(href, baseURL); ok {
						result.Links = append(result.Links, link)
					}
				}

			case tag == "input" && hasAttr:
				if strings.EqualFold(extractAttr(z, "type"), "password") {
					result.HasLoginForm = true
				}
			}

		case html.TextToken:
			if inTitle {
				result.Title = strings.TrimSpace(string(z.Text()))
				inTitle = false
			}

		case html.EndTagToken:
			tn, _ := z.TagName()
			if string(tn) == "title" {
				inTitle = false
			}
		}
	}
}

func isHeading(tag string) bool {
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

func extractAttr(z *html.Tokenizer, target string) string {
	for {
		key, val, more := z.TagAttr()
		if string(key) == target {
			return string(val)
		}
		if !more {
			return ""
		}
	}
}

func classifyLink(href string, baseURL *url.URL) (Link, bool) {
	parsed, err := url.Parse(href)
	if err != nil {
		return Link{}, false
	}

	resolved := baseURL.ResolveReference(parsed)

	// Skip non-http(s) schemes (mailto:, javascript:, tel:, etc.)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return Link{}, false
	}

	isInternal := strings.EqualFold(resolved.Host, baseURL.Host)
	return Link{URL: resolved.String(), IsInternal: isInternal}, true
}

func detectHTMLVersion(token html.Token) string {
	// The tokenizer stores the full doctype in token.Data.
	// HTML5: Data = "html"
	// Legacy: Data = `HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "..."`
	// https://www.w3.org/QA/2002/04/valid-dtd-list.html
	data := strings.ToLower(token.Data)

	if !strings.Contains(data, "public") {
		// HTML5 doctype has no PUBLIC identifier.
		return "HTML5"
	}

	switch {
	case strings.Contains(data, "xhtml 1.1") || strings.Contains(data, "xhtml basic 1.1"):
		return "XHTML 1.1"
	case strings.Contains(data, "xhtml 1.0"):
		return "XHTML 1.0"
	case strings.Contains(data, "html 4.01"):
		return "HTML 4.01"
	default:
		return "Unknown"
	}
}
