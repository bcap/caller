package plan

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// HTTP represents an HTTP call, describing both the request and which response to send
//
// There are 2 forms of defining request and/or response bodies:
//   - Explicitely defined by setting RequestBody or ResponseBody
//   - Randomly generated by setting GenRequestBody or GenResponseBody with the amount of
//     characters the body strings should have. Characters are regular ASCII single byte
//     runes. Check random.String documentation for more
//
// See also the HTTP.Parse function for creating HTTP structs from simple strings
type HTTP struct {
	Method          string            `json:"method" yaml:"method"`
	URL             URL               `json:"url" yaml:"url"`
	StatusCode      int               `json:"status-code" yaml:"status-code"`
	RequestBody     string            `json:"request-body,omitempty" yaml:"request-body,omitempty"`
	ResponseBody    string            `json:"response-body,omitempty" yaml:"response-body,omitempty"`
	GenRequestBody  int               `json:"gen-request-body,omitempty" yaml:"gen-request-body,omitempty"`
	GenResponseBody int               `json:"gen-response-body,omitempty" yaml:"gen-response-body,omitempty"`
	RequestHeaders  map[string]string `json:"request-headers,omitempty" yaml:"request-headers,omitempty"`
	ResponseHeaders map[string]string `json:"response-headers,omitempty" yaml:"response-headers,omitempty"`
}

func (h *HTTP) String() string {
	return fmt.Sprintf("%s %s %d", h.Method, h.URL.String(), h.StatusCode)
}

// The regex pattern used in the HTTP.Parse function
const HTTPPattern = `` +
	// Method
	`(\w+)\s+` +
	// URL
	`([^\s]+)\s+` +
	// Status code
	`(\d+)` +
	// Optional request body and response body sizes
	`(?:\s+(\d+)\s+(\d+))?`

var httpPattern = regexp.MustCompile(HTTPPattern)

// Parses a string into an HTTP object. Examples:
//   - Method, url and status code:
//     GET some/url?with=multiple&query=params 200
//   - Method, url, status code, request body size and response body size (both randomly generated):
//     POST some/url?with=multiple&query=params 200 500 1048
func (h *HTTP) Parse(s string) error {
	parts := httpPattern.FindStringSubmatch(s)
	if parts == nil {
		return fmt.Errorf("cannot parse http definition %q", s)
	}
	statusCode, err := strconv.Atoi(parts[3])
	if err != nil {
		return fmt.Errorf("cannot parse http definition %q: status code is not an integer: %w", s, err)
	}
	var reqBodySize int
	if parts[4] != "" {
		reqBodySize, err = strconv.Atoi(parts[4])
		if err != nil {
			return fmt.Errorf("cannot parse http definition %q: request body size is not an integer: %w", s, err)
		}
	}
	var respBodySize int
	if parts[4] != "" {
		respBodySize, err = strconv.Atoi(parts[5])
		if err != nil {
			return fmt.Errorf("cannot parse http definition %q: response body size is not an integer: %w", s, err)
		}
	}
	urlString := parts[2]
	if strings.Index(urlString, "http") != 0 {
		urlString = "http://" + urlString
	}
	url, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("cannot parse http definition %q: malformed url: %w", s, err)
	}
	h.Method = parts[1]
	h.URL = URL{URL: url}
	h.StatusCode = statusCode
	h.GenRequestBody = reqBodySize
	h.GenResponseBody = respBodySize
	return nil
}

func (h *HTTP) UnmarshalYAML(node *yaml.Node) error {
	if node.Value != "" {
		if err := h.Parse(node.Value); err != nil {
			return fmt.Errorf("invalid http definition at line %d: %w", node.Line, err)
		}
		return nil
	}
	type rawHTTP HTTP
	return node.Decode((*rawHTTP)(h))
}
