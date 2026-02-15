package pageinsight

import (
	"net/url"
	"strings"
	"testing"
)

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestParse_HTMLVersion(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "HTML5 lowercase",
			html:     `<!DOCTYPE html><html><head><title>Test</title></head><body></body></html>`,
			expected: "HTML5",
		},
		{
			name:     "HTML5 uppercase",
			html:     `<!DOCTYPE HTML><html><head><title>Test</title></head><body></body></html>`,
			expected: "HTML5",
		},
		{
			name:     "HTML 4.01 Strict",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd"><html><head><title>Test</title></head><body></body></html>`,
			expected: "HTML 4.01",
		},
		{
			name:     "HTML 4.01 Transitional",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd"><html><head><title>Test</title></head><body></body></html>`,
			expected: "HTML 4.01",
		},
		{
			name:     "HTML 4.01 Frameset",
			html:     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Frameset//EN" "http://www.w3.org/TR/html4/frameset.dtd"><html><head><title>Test</title></head><body></body></html>`,
			expected: "HTML 4.01",
		},
		{
			name:     "XHTML 1.0 Strict",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd"><html><head><title>Test</title></head><body></body></html>`,
			expected: "XHTML 1.0",
		},
		{
			name:     "XHTML 1.0 Transitional",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><html><head><title>Test</title></head><body></body></html>`,
			expected: "XHTML 1.0",
		},
		{
			name:     "XHTML 1.1",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"><html><head><title>Test</title></head><body></body></html>`,
			expected: "XHTML 1.1",
		},
		{
			name:     "XHTML Basic 1.1",
			html:     `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML Basic 1.1//EN" "http://www.w3.org/TR/xhtml-basic/xhtml-basic11.dtd"><html><head><title>Test</title></head><body></body></html>`,
			expected: "XHTML 1.1",
		},
		{
			name:     "no doctype",
			html:     `<html><head><title>Test</title></head><body></body></html>`,
			expected: "Unknown",
		},
	}

	base := mustParseURL("https://example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(strings.NewReader(tt.html), base)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.HTMLVersion != tt.expected {
				t.Errorf("HTMLVersion = %q, want %q", result.HTMLVersion, tt.expected)
			}
		})
	}
}

func TestParse_Title(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "simple title",
			html:     `<!DOCTYPE html><html><head><title>Hello World</title></head><body></body></html>`,
			expected: "Hello World",
		},
		{
			name:     "missing title",
			html:     `<!DOCTYPE html><html><head></head><body></body></html>`,
			expected: "",
		},
		{
			name:     "empty title",
			html:     `<!DOCTYPE html><html><head><title></title></head><body></body></html>`,
			expected: "",
		},
	}

	base := mustParseURL("https://example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(strings.NewReader(tt.html), base)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Title != tt.expected {
				t.Errorf("Title = %q, want %q", result.Title, tt.expected)
			}
		})
	}
}

func TestParse_Headings(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected map[string]int
	}{
		{
			name: "mixed headings",
			html: `<!DOCTYPE html><html><head><title>T</title></head><body>
			<h1>One</h1><h1>Two</h1><h2>Sub</h2><h3>Deep</h3>
			</body></html>`,
			expected: map[string]int{"h1": 2, "h2": 1, "h3": 1, "h4": 0, "h5": 0, "h6": 0},
		},
		{
			name:     "no headings",
			html:     `<!DOCTYPE html><html><head><title>T</title></head><body><p>text</p></body></html>`,
			expected: map[string]int{"h1": 0, "h2": 0, "h3": 0, "h4": 0, "h5": 0, "h6": 0},
		},
	}

	base := mustParseURL("https://example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(strings.NewReader(tt.html), base)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for level, count := range tt.expected {
				if result.Headings[level] != count {
					t.Errorf("Headings[%s] = %d, want %d", level, result.Headings[level], count)
				}
			}
		})
	}
}

func TestParse_Links(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>T</title></head><body>
	<a href="/about">About</a>
	<a href="https://example.com/contact">Contact</a>
	<a href="https://other.com/page">Other</a>
	<a href="mailto:test@example.com">Email</a>
	</body></html>`

	base := mustParseURL("https://example.com")
	result, err := Parse(strings.NewReader(html), base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var internal, external int
	for _, link := range result.Links {
		if link.IsInternal {
			internal++
		} else {
			external++
		}
	}

	// /about and /contact are internal (same host), /page is external
	// mailto is skipped (not http/https)
	if internal != 2 {
		t.Errorf("internal links = %d, want 2", internal)
	}
	if external != 1 {
		t.Errorf("external links = %d, want 1", external)
	}
}

func TestParse_LoginForm(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "has password input",
			html:     `<!DOCTYPE html><html><head><title>T</title></head><body><form><input type="password"></form></body></html>`,
			expected: true,
		},
		{
			name:     "no password input",
			html:     `<!DOCTYPE html><html><head><title>T</title></head><body><form><input type="text"></form></body></html>`,
			expected: false,
		},
		{
			name:     "no form at all",
			html:     `<!DOCTYPE html><html><head><title>T</title></head><body><p>Hello</p></body></html>`,
			expected: false,
		},
	}

	base := mustParseURL("https://example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(strings.NewReader(tt.html), base)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.HasLoginForm != tt.expected {
				t.Errorf("HasLoginForm = %v, want %v", result.HasLoginForm, tt.expected)
			}
		})
	}
}
