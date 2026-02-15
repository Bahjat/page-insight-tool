package model

// PageAnalysis holds the complete result of analyzing a web page.
type PageAnalysis struct {
	URL          string         `json:"url"`
	HTMLVersion  string         `json:"html_version"`
	Title        string         `json:"title"`
	Headings     map[string]int `json:"headings"`
	Links        LinkStats      `json:"links"`
	HasLoginForm bool           `json:"has_login_form"`
}

// LinkStats breaks down the links found on a page.
type LinkStats struct {
	Internal     int `json:"internal_count"`
	External     int `json:"external_count"`
	Inaccessible int `json:"inaccessible_count"`
}

// ErrorResponse is the JSON shape returned on failure.
type ErrorResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}
