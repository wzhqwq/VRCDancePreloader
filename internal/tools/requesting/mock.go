package requesting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// MockEnvVar is the environment variable read by ApplyMockFromEnv. When it holds
// the path of a mock rule file (see LoadMockRules) that mock is installed; when
// it is unset or empty nothing changes.
//
// Nothing in the production start path reads it: whoever wants the mock (a test,
// or a debug build) has to ask for it. That is what keeps the rules engine out of
// a release binary, so do not add a call to initialize().
const MockEnvVar = "VRCDP_MOCK"

// MockRule describes one mocked HTTP response.
//
// Matching: Method, Host, Path, PathPrefix and PathRegexp are ANDed together;
// an empty field means "any". Rules are evaluated in order and the first match
// wins. A request that matches no rule fails with
// "requesting: no mock rule for METHOD URL" and is never forwarded to the real
// network.
type MockRule struct {
	Name   string // optional, appears in matching errors and log messages
	Method string // "" = any
	Host   string // "api.github.com" | "*.githubusercontent.com" | "" = any

	Path       string // exact match on URL path; "" = any
	PathPrefix string // prefix match on URL path; "" = any
	PathRegexp string // Go regexp matched against URL path; "" = any

	Status   int           // 0 => 200
	Header   http.Header   // response headers (Content-Type etc.)
	Body     []byte        // response payload
	BodyJSON any           // json.Marshal'ed into the body; replaces Body and defaults Content-Type to application/json
	Range    bool          // honour the request's Range header: 206 + Content-Range, 416 when out of range
	Latency  time.Duration // artificial delay before responding; 0 = no wait

	// Handler is the escape hatch: when set it fully owns the response and all
	// of the fields above (except matching) are ignored. Unlike the fields
	// above, the handler is responsible for filling StatusCode/Status/Header/
	// Body/ContentLength/Request itself.
	Handler func(*http.Request) (*http.Response, error)
}

// MockTransport is an http.RoundTripper that answers requests from a fixed list
// of MockRule. It holds no real transport at all, so it is structurally unable
// to reach the network, and an unmatched request is reported as an error
// instead of being forwarded.
//
// It is safe for concurrent use.
type MockTransport struct {
	mu     sync.Mutex
	rules  []preparedRule // immutable after construction; safe to read without the lock
	served []*http.Request
	misses []string
}

type preparedRule struct {
	rule   MockRule
	pathRe *regexp.Regexp
	body   []byte
	header http.Header
}

// NewMockTransport builds a transport from rules. Rules are matched in order,
// the first match wins; an unmatched request returns an error.
//
// A rule with an invalid PathRegexp (or a BodyJSON that cannot be marshalled)
// panics: this constructor cannot return an error. LoadMockRules reports both
// as regular errors instead.
func NewMockTransport(rules ...MockRule) *MockTransport {
	m := &MockTransport{}
	for i, r := range rules {
		m.rules = append(m.rules, prepareRule(i, r))
	}
	return m
}

// Requests returns every request that reached this transport, including the
// ones that matched no rule.
func (m *MockTransport) Requests() []*http.Request {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*http.Request, len(m.served))
	copy(out, m.served)
	return out
}

// Unmatched returns the "METHOD URL" of every request that matched no rule.
func (m *MockTransport) Unmatched() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]string, len(m.misses))
	copy(out, m.misses)
	return out
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	m.served = append(m.served, req)
	// m.rules is immutable, and the transport never touches a real
	// RoundTripper, so a request can only be answered or rejected here.
	rules := m.rules
	m.mu.Unlock()

	for i := range rules {
		if !rules[i].matches(req) {
			continue
		}
		return rules[i].respond(req)
	}

	key := req.Method + " " + req.URL.String()

	m.mu.Lock()
	m.misses = append(m.misses, key)
	m.mu.Unlock()

	return nil, fmt.Errorf("requesting: no mock rule for %s", key)
}

func prepareRule(index int, r MockRule) preparedRule {
	p := preparedRule{rule: r, body: r.Body, header: cloneHeader(r.Header)}

	if r.PathRegexp != "" {
		re, err := regexp.Compile(r.PathRegexp)
		if err != nil {
			panic(fmt.Sprintf("requesting: mock rule #%d (%s): invalid pathRegexp %q: %v", index, r.label(), r.PathRegexp, err))
		}
		p.pathRe = re
	}

	if r.BodyJSON != nil {
		b, err := json.Marshal(r.BodyJSON)
		if err != nil {
			panic(fmt.Sprintf("requesting: mock rule #%d (%s): cannot marshal bodyJson: %v", index, r.label(), err))
		}
		p.body = b
		if p.header.Get("Content-Type") == "" {
			p.header.Set("Content-Type", "application/json")
		}
	}

	return p
}

func (r MockRule) label() string {
	if r.Name != "" {
		return r.Name
	}
	return "unnamed"
}

func (p *preparedRule) matches(req *http.Request) bool {
	r := p.rule

	if r.Method != "" && !strings.EqualFold(r.Method, req.Method) {
		return false
	}
	if !matchMockHost(r.Host, req.URL.Hostname()) {
		return false
	}
	if r.Path != "" && req.URL.Path != r.Path {
		return false
	}
	if r.PathPrefix != "" && !strings.HasPrefix(req.URL.Path, r.PathPrefix) {
		return false
	}
	if p.pathRe != nil && !p.pathRe.MatchString(req.URL.Path) {
		return false
	}

	return true
}

// matchMockHost implements the Host predicate: "" matches anything, an exact
// host matches itself (case-insensitively), and a "*.suffix" pattern matches
// both "suffix" and any subdomain of it.
func matchMockHost(pattern, host string) bool {
	if pattern == "" {
		return true
	}
	if suffix, ok := strings.CutPrefix(pattern, "*."); ok {
		if strings.EqualFold(host, suffix) {
			return true
		}
		return strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(suffix))
	}
	return strings.EqualFold(pattern, host)
}

func (p *preparedRule) respond(req *http.Request) (*http.Response, error) {
	if p.rule.Handler != nil {
		return p.rule.Handler(req)
	}

	if p.rule.Latency > 0 {
		timer := time.NewTimer(p.rule.Latency)
		select {
		case <-timer.C:
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		}
	}

	status := p.rule.Status
	if status == 0 {
		status = http.StatusOK
	}
	body := p.body
	header := cloneHeader(p.header)

	if p.rule.Range {
		if spec := req.Header.Get("Range"); spec != "" {
			if start, end, ok := parseByteRange(spec, int64(len(body))); ok {
				if start > end || start >= int64(len(body)) {
					header.Set("Content-Range", fmt.Sprintf("bytes */%d", len(body)))
					return newMockResponse(req, http.StatusRequestedRangeNotSatisfiable, header, nil, 0), nil
				}
				header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
				return newMockResponse(req, http.StatusPartialContent, header, body[start:end+1], end-start+1), nil
			}
		}
	}

	if req.Method == http.MethodHead || status == http.StatusNoContent || status == http.StatusNotModified {
		// Mirror real HTTP: these responses carry no payload even though the
		// declared length still describes what a body-carrying response would
		// contain.
		return newMockResponse(req, status, header, nil, int64(len(body))), nil
	}

	return newMockResponse(req, status, header, body, int64(len(body))), nil
}

// newMockResponse builds a fully populated *http.Response. A custom
// RoundTripper has to fill these fields itself: http.Client only fills in the
// ones it owns (and only for a nil Body), so downstream code such as
// api/github.go (res.Status) or task/remotes.go (res.ContentLength) would
// otherwise read zero values.
func newMockResponse(req *http.Request, status int, header http.Header, body []byte, contentLength int64) *http.Response {
	if status == 0 {
		status = http.StatusOK
	}
	if header == nil {
		header = http.Header{}
	}

	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),

		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,

		Header: header,

		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: contentLength,

		Request: req,
	}
}

// parseByteRange parses a single-range "bytes=" header. ok is false when the
// header cannot be understood (which means: serve the full body). A satisfiable
// range with start > end (e.g. an empty body) is reported as ok with
// start > end, and the caller turns it into 416.
func parseByteRange(spec string, size int64) (start, end int64, ok bool) {
	value, found := strings.CutPrefix(strings.TrimSpace(spec), "bytes=")
	if !found {
		return 0, 0, false
	}
	value = strings.TrimSpace(value)
	if strings.Contains(value, ",") {
		// multi-range requests are served as the full body
		return 0, 0, false
	}

	first, last, found := strings.Cut(value, "-")
	if !found {
		return 0, 0, false
	}
	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)

	if first == "" {
		// suffix range: the last N bytes
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		if n > size {
			n = size
		}
		return size - n, size - 1, true
	}

	from, err := strconv.ParseInt(first, 10, 64)
	if err != nil || from < 0 {
		return 0, 0, false
	}
	if last == "" {
		return from, size - 1, true
	}

	to, err := strconv.ParseInt(last, 10, 64)
	if err != nil || to < from {
		return 0, 0, false
	}
	if to > size-1 {
		to = size - 1
	}
	return from, to, true
}

func cloneHeader(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, values := range h {
		cp := make([]string, len(values))
		copy(cp, values)
		out[k] = cp
	}
	return out
}

// MockAvailabilityRules returns one rule per availability self-test case
// (derived from testCases), so that every self-test started by initialize()
// passes while a mock is installed. A test that needs the availability gates to
// be open typically does:
//
//	requesting.SetMock(requesting.NewMockTransport(append(requesting.MockAvailabilityRules(), myRules...)...))
//
// Note that the rule for https://api.github.com and https://release-assets.githubusercontent.com
// only matches the site root, so rules for other paths on those hosts still
// apply.
func MockAvailabilityRules() []MockRule {
	names := make([]ClientName, 0, len(testCases))
	for name := range testCases {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })

	rules := make([]MockRule, 0, len(names))
	for _, name := range names {
		tc := testCases[name]
		if tc.url == "" {
			continue
		}
		u, err := url.Parse(tc.url)
		if err != nil {
			continue
		}

		rule := MockRule{
			Name:   "availability:" + string(name),
			Method: mockMethod(tc),
			Host:   u.Hostname(),
			Status: tc.expectedStatus,
		}
		if u.Path == "" {
			// an exact match on the root; "Path: ''" would mean "any path"
			rule.PathRegexp = `^/?$`
		} else {
			rule.Path = u.Path
		}

		if tc.minContentLength > 0 {
			// availability.go requires a body of at least minContentLength
			// bytes, and the check reads http.Response.ContentLength.
			rule.Body = make([]byte, tc.minContentLength)
		} else {
			rule.Body = []byte("mocked availability response")
		}

		rules = append(rules, rule)
	}

	return rules
}

func mockMethod(tc testCase) string {
	if tc.useGet {
		return http.MethodGet
	}
	return http.MethodHead
}

// ApplyMockFromEnv installs the mock described by the VRCDP_MOCK environment
// variable (a mock rule file, see LoadMockRules). It returns (nil, nil) when the
// variable is unset or empty.
//
// It is deliberately not called by the production start path (initialize): the
// mock is a test / offline-debugging facility, and leaving the call out of
// initialize is what lets the linker drop this file's rules engine from a release
// binary. Tests and debug builds call it themselves.
func ApplyMockFromEnv() (*MockTransport, error) {
	path := strings.TrimSpace(os.Getenv(MockEnvVar))
	if path == "" {
		return nil, nil
	}

	m, err := LoadMockRules(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", MockEnvVar, err)
	}

	SetMock(m)
	return m, nil
}

// mockRuleFile is the YAML shape of a mock rule file:
//
//	rules:
//	  - name: github-latest-release
//	    method: GET
//	    host: api.github.com
//	    path: /repos/o/r/releases/latest
//	    status: 200
//	    header:
//	      Content-Type: application/json
//	    body: |
//	      {"tag_name":"v1.2.3"}
//	    # bodyFile: testdata/release.json   (relative to the rule file)
//	    # range: true
//	    # latency: 10ms
type mockRuleFile struct {
	Rules []mockRuleYAML `yaml:"rules"`
}

type mockRuleYAML struct {
	Name   string `yaml:"name"`
	Method string `yaml:"method"`
	Host   string `yaml:"host"`

	Path       string `yaml:"path"`
	PathPrefix string `yaml:"pathPrefix"`
	PathRegexp string `yaml:"pathRegexp"`

	Status   int               `yaml:"status"`
	Header   map[string]string `yaml:"header"`
	Body     string            `yaml:"body"`
	BodyFile string            `yaml:"bodyFile"`
	Range    bool              `yaml:"range"`
	Latency  time.Duration     `yaml:"latency"`
}

// LoadMockRules reads a YAML rule file and builds a MockTransport from it. A
// relative bodyFile is resolved against the directory of the rule file. Every
// error is annotated with the rule name and its position in the file.
func LoadMockRules(path string) (*MockTransport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read mock rule file: %w", err)
	}

	var file mockRuleFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("cannot parse mock rule file %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	rules := make([]MockRule, 0, len(file.Rules))
	for i, raw := range file.Rules {
		rule, err := raw.toRule(dir)
		if err != nil {
			return nil, fmt.Errorf("mock rule file %s: rule #%d (%s): %w", path, i, raw.label(), err)
		}
		rules = append(rules, rule)
	}

	return NewMockTransport(rules...), nil
}

func (raw mockRuleYAML) label() string {
	if raw.Name != "" {
		return raw.Name
	}
	return "unnamed"
}

func (raw mockRuleYAML) toRule(dir string) (MockRule, error) {
	rule := MockRule{
		Name:       raw.Name,
		Method:     raw.Method,
		Host:       raw.Host,
		Path:       raw.Path,
		PathPrefix: raw.PathPrefix,
		PathRegexp: raw.PathRegexp,
		Status:     raw.Status,
		Range:      raw.Range,
		Latency:    raw.Latency,
	}

	if rule.PathRegexp != "" {
		if _, err := regexp.Compile(rule.PathRegexp); err != nil {
			return MockRule{}, fmt.Errorf("invalid pathRegexp %q: %w", rule.PathRegexp, err)
		}
	}

	if len(raw.Header) > 0 {
		header := http.Header{}
		for key, value := range raw.Header {
			header.Set(key, value)
		}
		rule.Header = header
	}

	if raw.Body != "" && raw.BodyFile != "" {
		return MockRule{}, errors.New("body and bodyFile are mutually exclusive")
	}

	switch {
	case raw.BodyFile != "":
		bodyPath := raw.BodyFile
		if !filepath.IsAbs(bodyPath) {
			bodyPath = filepath.Join(dir, bodyPath)
		}
		body, err := os.ReadFile(bodyPath)
		if err != nil {
			return MockRule{}, fmt.Errorf("cannot read bodyFile %q: %w", raw.BodyFile, err)
		}
		rule.Body = body
	case raw.Body != "":
		rule.Body = []byte(raw.Body)
	}

	return rule, nil
}
