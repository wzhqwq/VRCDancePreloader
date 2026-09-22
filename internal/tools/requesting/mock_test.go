package requesting

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/api/youtube/v3"
)

// deadEndpoint is a loopback address nothing listens on. Tests use it as the
// mock host so that a request escaping the mock fails immediately with
// "connection refused" instead of touching the real network.
const deadEndpoint = "http://127.0.0.1:9"

var mockTestClients = []ClientName{
	NoProxy, PyPyDance, WannaDance, DuDuFitDance, BiliBili,
	YouTubeVideo, YouTubeApi, YouTubeImage, GitHubApi, GitHubAssets,
}

func newTestProvider() *ClientProvider {
	return NewProxyProvider("", "mock_test", testCase{})
}

// doGet performs a GET through p and returns the raw outcome, so that tests can
// assert on both the success and the error path.
func doGet(p *ClientProvider, rawURL string, header map[string]string) (*http.Response, error) {
	req, err := p.NewGetRequest(rawURL, context.Background())
	if err != nil {
		return nil, err
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	return p.Do(req)
}

func mustGet(t *testing.T, p *ClientProvider, rawURL string, header map[string]string) (*http.Response, string) {
	t.Helper()

	resp, err := doGet(p, rawURL, header)
	if err != nil {
		t.Fatalf("GET %s: unexpected error: %v", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("GET %s: reading body: %v", rawURL, err)
	}
	return resp, string(body)
}

func sortedTestCaseNames() []ClientName {
	names := make([]ClientName, 0, len(testCases))
	for name := range testCases {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// M1: rules are matched in order (first match wins) and the method / host
// wildcard / path / pathPrefix predicates are honoured.
func TestMockM1RuleOrder(t *testing.T) {
	t.Run("first match wins", func(t *testing.T) {
		SetMock(NewMockTransport(
			MockRule{Name: "first", Method: "GET", Host: "order.test", Path: "/x", Status: 201, Body: []byte("first")},
			MockRule{Name: "second", Method: "GET", Host: "order.test", Path: "/x", Status: 202, Body: []byte("second")},
		))
		t.Cleanup(ClearMock)

		resp, body := mustGet(t, newTestProvider(), "http://order.test/x", nil)
		if resp.StatusCode != 201 || body != "first" {
			t.Fatalf("M1: the first matching rule must win: got status=%d body=%q, want status=201 body=\"first\"", resp.StatusCode, body)
		}
	})

	t.Run("host wildcard", func(t *testing.T) {
		SetMock(NewMockTransport(MockRule{Name: "wild", Host: "*.wild.test", Path: "/y", Body: []byte("wild")}))
		t.Cleanup(ClearMock)

		p := newTestProvider()
		if _, body := mustGet(t, p, "http://cdn.wild.test/y", nil); body != "wild" {
			t.Fatalf("M1: a subdomain must match *.wild.test: body=%q, want \"wild\"", body)
		}
		if _, body := mustGet(t, p, "http://wild.test/y", nil); body != "wild" {
			t.Fatalf("M1: the bare domain must match *.wild.test: body=%q, want \"wild\"", body)
		}
		if resp, err := doGet(p, "http://other.test/y", nil); err == nil {
			t.Fatalf("M1: other.test must not match *.wild.test, but the request was served with status=%d", resp.StatusCode)
		}
	})

	t.Run("path prefix and exact path", func(t *testing.T) {
		SetMock(NewMockTransport(
			MockRule{Name: "prefix", Host: "prefix.test", PathPrefix: "/api/v1/", Body: []byte("prefix")},
			MockRule{Name: "exact", Host: "exact.test", Path: "/api/v1/exact", Body: []byte("exact")},
		))
		t.Cleanup(ClearMock)

		p := newTestProvider()
		if _, body := mustGet(t, p, "http://prefix.test/api/v1/deep/thing", nil); body != "prefix" {
			t.Fatalf("M1: pathPrefix must match /api/v1/deep/thing: body=%q, want \"prefix\"", body)
		}
		if resp, err := doGet(p, "http://prefix.test/api/v2/thing", nil); err == nil {
			t.Fatalf("M1: PathPrefix /api/v1/ must not match /api/v2/thing, got status=%d", resp.StatusCode)
		}
		if _, body := mustGet(t, p, "http://exact.test/api/v1/exact", nil); body != "exact" {
			t.Fatalf("M1: Path must match /api/v1/exact exactly: body=%q, want \"exact\"", body)
		}
		if resp, err := doGet(p, "http://exact.test/api/v1/exact/more", nil); err == nil {
			t.Fatalf("M1: Path /api/v1/exact must not match /api/v1/exact/more, got status=%d", resp.StatusCode)
		}
	})

	t.Run("method and path must match", func(t *testing.T) {
		m := NewMockTransport(MockRule{Name: "get-only", Method: "GET", Host: "method.test", Path: "/m", Body: []byte("m")})
		SetMock(m)
		t.Cleanup(ClearMock)

		p := newTestProvider()
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://method.test/m", nil)
		if err != nil {
			t.Fatalf("M1: building the POST request: %v", err)
		}
		resp, err := p.Do(req)
		if err == nil {
			t.Fatalf("M1: a POST must not match a GET-only rule, got status=%d", resp.StatusCode)
		}
		if want := "requesting: no mock rule for POST http://method.test/m"; !strings.Contains(err.Error(), want) {
			t.Fatalf("M1: error = %q, want it to contain %q", err.Error(), want)
		}
	})
}

// M2: Range support — 206 + Content-Range + ContentLength, 416 out of range,
// 200 without a Range header.
func TestMockM2Range(t *testing.T) {
	payload := []byte("0123456789")

	SetMock(NewMockTransport(MockRule{Name: "range", Host: "range.test", Path: "/file", Body: payload, Range: true}))
	t.Cleanup(ClearMock)

	p := newTestProvider()

	t.Run("open ended range", func(t *testing.T) {
		resp, body := mustGet(t, p, "http://range.test/file", map[string]string{"Range": "bytes=0-"})
		if resp.StatusCode != http.StatusPartialContent {
			t.Fatalf("M2: Range bytes=0- must answer 206, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Content-Range"); got != "bytes 0-9/10" {
			t.Fatalf("M2: Content-Range = %q, want \"bytes 0-9/10\"", got)
		}
		if resp.ContentLength != 10 {
			t.Fatalf("M2: ContentLength = %d, want 10", resp.ContentLength)
		}
		if body != "0123456789" {
			t.Fatalf("M2: body = %q, want \"0123456789\"", body)
		}
	})

	t.Run("offset range", func(t *testing.T) {
		resp, body := mustGet(t, p, "http://range.test/file", map[string]string{"Range": "bytes=4-"})
		if resp.StatusCode != http.StatusPartialContent {
			t.Fatalf("M2: Range bytes=4- must answer 206, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Content-Range"); got != "bytes 4-9/10" {
			t.Fatalf("M2: Content-Range = %q, want \"bytes 4-9/10\"", got)
		}
		if resp.ContentLength != 6 {
			t.Fatalf("M2: ContentLength = %d, want 6 (the remaining bytes)", resp.ContentLength)
		}
		if body != "456789" {
			t.Fatalf("M2: body = %q, want \"456789\"", body)
		}
	})

	t.Run("closed range", func(t *testing.T) {
		resp, body := mustGet(t, p, "http://range.test/file", map[string]string{"Range": "bytes=2-5"})
		if resp.StatusCode != http.StatusPartialContent {
			t.Fatalf("M2: Range bytes=2-5 must answer 206, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Content-Range"); got != "bytes 2-5/10" {
			t.Fatalf("M2: Content-Range = %q, want \"bytes 2-5/10\"", got)
		}
		if resp.ContentLength != 4 || body != "2345" {
			t.Fatalf("M2: got ContentLength=%d body=%q, want ContentLength=4 body=\"2345\"", resp.ContentLength, body)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		resp, body := mustGet(t, p, "http://range.test/file", map[string]string{"Range": "bytes=10-"})
		if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
			t.Fatalf("M2: Range bytes=10- must answer 416, got %d", resp.StatusCode)
		}
		if resp.ContentLength != 0 || body != "" {
			t.Fatalf("M2: a 416 must have an empty body, got ContentLength=%d body=%q", resp.ContentLength, body)
		}
	})

	t.Run("without range header", func(t *testing.T) {
		resp, body := mustGet(t, p, "http://range.test/file", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("M2: no Range header must answer 200, got %d", resp.StatusCode)
		}
		if resp.ContentLength != 10 || body != "0123456789" {
			t.Fatalf("M2: got ContentLength=%d body=%q, want ContentLength=10 body=\"0123456789\"", resp.ContentLength, body)
		}
		if got := resp.Header.Get("Content-Range"); got != "" {
			t.Fatalf("M2: a 200 must not carry Content-Range, got %q", got)
		}
	})

	t.Run("range disabled", func(t *testing.T) {
		SetMock(NewMockTransport(MockRule{Name: "plain", Host: "plain.test", Path: "/file", Body: payload}))
		t.Cleanup(ClearMock)

		resp, body := mustGet(t, newTestProvider(), "http://plain.test/file", map[string]string{"Range": "bytes=4-"})
		if resp.StatusCode != http.StatusOK || body != "0123456789" {
			t.Fatalf("M2: Range=false must serve the full body with 200, got status=%d body=%q", resp.StatusCode, body)
		}
	})
}

// M3: an unmatched request must be reported by the mock and never fall back to
// the real network.
func TestMockM3UnmatchedNeverReachesNetwork(t *testing.T) {
	const target = "https://no-such-host.invalid/x"

	m := NewMockTransport(MockRule{Name: "known", Host: "known.test", Path: "/known", Body: []byte("known")})
	SetMock(m)
	t.Cleanup(ClearMock)

	resp, err := doGet(newTestProvider(), target, nil)
	if resp != nil {
		t.Fatalf("M3: an unmatched request must not produce a response, got status=%d", resp.StatusCode)
	}
	if err == nil {
		t.Fatal("M3: an unmatched request must return an error")
	}

	want := "requesting: no mock rule for GET " + target
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("M3: error = %q, want it to contain %q", err.Error(), want)
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		t.Fatalf("M3: the request escaped the mock and reached the network: %v", err)
	}

	unmatched := m.Unmatched()
	if len(unmatched) != 1 || unmatched[0] != "GET "+target {
		t.Fatalf("M3: Unmatched() = %v, want [\"GET %s\"]", unmatched, target)
	}
}

// M4: every ClientName (NoProxy included) is wired through a gate, so a mock
// installed with SetMock serves all of them.
func TestMockM4EveryClientUsesMock(t *testing.T) {
	m := NewMockTransport(MockRule{Name: "local", Host: "127.0.0.1", Body: []byte("mocked")})
	SetMock(m)
	t.Cleanup(ClearMock)

	for _, name := range mockTestClients {
		if clients[name] == nil {
			initClient(name, "", false)
		}

		p := clients[name]
		resp, err := doGet(p, deadEndpoint+"/hit", nil)
		if err != nil {
			t.Fatalf("M4: client %q did not serve the request from the mock: %v", name, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("M4: client %q: reading body: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK || string(body) != "mocked" {
			t.Fatalf("M4: client %q got status=%d body=%q, want status=200 body=\"mocked\"", name, resp.StatusCode, string(body))
		}
	}

	if got := len(m.Requests()); got != len(mockTestClients) {
		t.Fatalf("M4: the mock served %d requests, want %d (one per ClientName)", got, len(mockTestClients))
	}
}

// M5: the YouTube Data API path — the only exit that calls
// Client.Transport.RoundTrip directly — is served by the mock, and switching the
// YouTubeApi client to "no proxy" no longer leaves a nil Transport.
func TestMockM5YoutubeApiUsesMock(t *testing.T) {
	m := NewMockTransport(MockRule{
		Name:   "youtube-videos-list",
		Method: "GET",
		// google.golang.org/api resolves its base path to youtube.googleapis.com
		// (www.googleapis.com in older versions), so match both.
		Host: "*.googleapis.com",
		Path: "/youtube/v3/videos",
		BodyJSON: map[string]any{
			"items": []any{
				map[string]any{
					"snippet":        map[string]any{"title": "Mocked Video", "channelTitle": "Mock Channel"},
					"contentDetails": map[string]any{"duration": "PT1M30S"},
				},
			},
		},
	})
	SetMock(m)
	t.Cleanup(ClearMock)

	// A configured proxy followed by a hot switch to "no proxy": this is the path
	// that used to end up with &http.Client{} (nil Transport).
	initClient(YouTubeApi, deadEndpoint, false)
	clients[YouTubeApi].SetProxy("")

	svc, err := youtube.NewService(context.Background(), WithYoutubeApiClient("test-api-key"))
	if err != nil {
		t.Fatalf("M5: youtube.NewService: %v", err)
	}

	list, err := svc.Videos.List([]string{"snippet"}).Id("mock-video").Do()
	if err != nil {
		t.Fatalf("M5: Videos.List().Do() did not go through the mock: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Snippet == nil {
		t.Fatalf("M5: unexpected mocked payload: %+v", list.Items)
	}
	if got := list.Items[0].Snippet.Title; got != "Mocked Video" {
		t.Fatalf("M5: snippet title = %q, want \"Mocked Video\"", got)
	}

	requests := m.Requests()
	if len(requests) != 1 {
		t.Fatalf("M5: the mock served %d requests, want 1", len(requests))
	}
	if got := requests[0].URL.Path; got != "/youtube/v3/videos" {
		t.Fatalf("M5: requested path = %q, want \"/youtube/v3/videos\"", got)
	}
	if got := requests[0].URL.Hostname(); !strings.HasSuffix(got, ".googleapis.com") {
		t.Fatalf("M5: requested host = %q, want a googleapis.com host", got)
	}
	if got := requests[0].URL.Query().Get("key"); got != "test-api-key" {
		t.Fatalf("M5: mixedTransport must inject the API key, got key=%q", got)
	}
}

// M6: clients created after SetMock (initClient, SetProxy, a new provider) pick
// the mock up as well.
func TestMockM6LateClientsGetMock(t *testing.T) {
	m := NewMockTransport(MockRule{Name: "local", Host: "127.0.0.1", Body: []byte("mocked-late")})
	SetMock(m)
	t.Cleanup(ClearMock)

	check := func(t *testing.T, label string, p *ClientProvider) {
		t.Helper()

		resp, body := mustGet(t, p, deadEndpoint+"/hit", nil)
		if resp.StatusCode != http.StatusOK || body != "mocked-late" {
			t.Fatalf("M6: %s got status=%d body=%q, want status=200 body=\"mocked-late\"", label, resp.StatusCode, body)
		}
	}

	initClient(BiliBili, deadEndpoint, false)
	check(t, "a client created by initClient after SetMock", clients[BiliBili])

	clients[BiliBili].SetProxy("")
	check(t, "a client replaced by SetProxy after SetMock", clients[BiliBili])

	check(t, "a provider created by NewProxyProvider after SetMock", newTestProvider())

	if got := len(m.Requests()); got != 3 {
		t.Fatalf("M6: the mock served %d requests, want 3", got)
	}
}

// M7: MockAvailabilityRules() makes all nine availability self-tests pass.
func TestMockM7AvailabilityRules(t *testing.T) {
	rules := MockAvailabilityRules()
	if len(rules) != len(testCases) {
		t.Fatalf("M7: MockAvailabilityRules() returned %d rules, want one per test case (%d)", len(rules), len(testCases))
	}

	SetMock(NewMockTransport(rules...))
	t.Cleanup(ClearMock)

	for _, name := range sortedTestCaseNames() {
		tc := testCases[name]
		p := NewProxyProvider("", string(name), tc)
		if err := testClient(p.Client(), string(name), tc, ""); err != nil {
			t.Errorf("M7: availability self-test for %q failed with MockAvailabilityRules() installed: %v", name, err)
		}
	}
}

// M8: the rule file format (body / bodyFile / header / wildcard / relative
// path / latency) and its error reporting.
func TestMockM8LoadMockRules(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "release.json"), `{"tag_name":"v9.9.9"}`)

	ruleFile := filepath.Join(dir, "mock.yaml")
	writeTestFile(t, ruleFile, `
rules:
  - name: inline-body
    method: GET
    host: api.github.com
    path: /inline
    status: 201
    header:
      Content-Type: application/json
    body: |
      {"tag_name":"v1.2.3"}
  - name: body-from-file
    method: GET
    host: api.github.com
    path: /from-file
    bodyFile: release.json
  - name: wildcard-prefix
    host: "*.wild.test"
    pathPrefix: /assets/
    status: 206
    latency: 20ms
    body: wild
  - name: regexp-path
    host: re.test
    pathRegexp: "^/v[0-9]+/thing$"
    body: regexp-body
`)

	m, err := LoadMockRules(ruleFile)
	if err != nil {
		t.Fatalf("M8: LoadMockRules(%s): %v", ruleFile, err)
	}
	SetMock(m)
	t.Cleanup(ClearMock)

	p := newTestProvider()

	resp, body := mustGet(t, p, "http://api.github.com/inline", nil)
	if resp.StatusCode != 201 {
		t.Fatalf("M8: inline rule status = %d, want 201", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("M8: inline rule Content-Type = %q, want \"application/json\"", got)
	}
	if strings.TrimSpace(body) != `{"tag_name":"v1.2.3"}` {
		t.Fatalf("M8: inline rule body = %q, want the YAML literal block", body)
	}

	if _, body := mustGet(t, p, "http://api.github.com/from-file", nil); strings.TrimSpace(body) != `{"tag_name":"v9.9.9"}` {
		t.Fatalf("M8: bodyFile body = %q, want the content of release.json (resolved relative to the rule file)", body)
	}

	start := time.Now()
	resp, body = mustGet(t, p, "http://cdn.wild.test/assets/thing", nil)
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("M8: latency: 20ms must be honoured, the request took %v", elapsed)
	}
	if resp.StatusCode != 206 || body != "wild" {
		t.Fatalf("M8: wildcard+prefix rule got status=%d body=%q, want status=206 body=\"wild\"", resp.StatusCode, body)
	}

	if _, body := mustGet(t, p, "http://re.test/v42/thing", nil); body != "regexp-body" {
		t.Fatalf("M8: pathRegexp rule body = %q, want \"regexp-body\"", body)
	}
	if resp, err := doGet(p, "http://re.test/v42/other", nil); err == nil {
		t.Fatalf("M8: pathRegexp must not match /v42/other, got status=%d", resp.StatusCode)
	}

	broken := []struct {
		label   string
		content string
		want    string
	}{
		{
			label: "body and bodyFile together",
			content: `rules:
  - name: both
    host: both.test
    body: inline
    bodyFile: release.json
`,
			want: "both",
		},
		{
			label: "missing bodyFile",
			content: `rules:
  - name: missing-file
    host: missing.test
    bodyFile: nope.json
`,
			want: "missing-file",
		},
		{
			label: "invalid pathRegexp",
			content: `rules:
  - name: bad-regexp
    host: bad.test
    pathRegexp: "(["
`,
			want: "bad-regexp",
		},
		{
			label:   "broken yaml",
			content: "rules:\n  - name: [\n",
			want:    "",
		},
	}

	for i, c := range broken {
		path := filepath.Join(dir, "broken.yaml")
		writeTestFile(t, path, c.content)

		_, err := LoadMockRules(path)
		if err == nil {
			t.Fatalf("M8: %s must be rejected, but LoadMockRules returned no error", c.label)
		}
		if c.want != "" && !strings.Contains(err.Error(), c.want) {
			t.Fatalf("M8: %s: error %q must name the rule %q", c.label, err.Error(), c.want)
		}
		if i == 0 && !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("M8: body + bodyFile error = %q, want it to mention the conflict", err.Error())
		}
	}
}

// M9: ApplyMockFromEnv is a no-op without VRCDP_MOCK and installs the mock of
// the referenced file otherwise.
func TestMockM9ApplyMockFromEnv(t *testing.T) {
	dir := t.TempDir()
	ruleFile := filepath.Join(dir, "mock.yaml")
	writeTestFile(t, ruleFile, `rules:
  - name: from-env
    host: env.test
    path: /x
    body: from-env
`)
	missing := filepath.Join(dir, "missing.yaml")

	ClearMock()
	t.Cleanup(ClearMock)

	t.Setenv(MockEnvVar, "")
	if m, err := ApplyMockFromEnv(); err != nil || m != nil || Mock() != nil {
		t.Fatalf("M9: an unset %s must be a no-op: transport=%v err=%v Mock()=%v", MockEnvVar, m, err, Mock())
	}

	t.Setenv(MockEnvVar, ruleFile)
	m, err := ApplyMockFromEnv()
	if err != nil {
		t.Fatalf("M9: ApplyMockFromEnv() with %s=%s: %v", MockEnvVar, ruleFile, err)
	}
	if m == nil || Mock() != m {
		t.Fatalf("M9: ApplyMockFromEnv() must install the mock: transport=%v Mock()=%v", m, Mock())
	}
	if _, body := mustGet(t, newTestProvider(), "http://env.test/x", nil); body != "from-env" {
		t.Fatalf("M9: the installed mock must serve the rule file: body=%q, want \"from-env\"", body)
	}

	t.Setenv(MockEnvVar, missing)
	if _, err := ApplyMockFromEnv(); err == nil {
		t.Fatalf("M9: %s pointing at a missing file must be reported", MockEnvVar)
	}
	if Mock() != m {
		t.Fatalf("M9: a failed ApplyMockFromEnv must leave the installed mock untouched")
	}
}

// M11: the production start path must not read VRCDP_MOCK.
//
// Measured (see plans/20260920-requesting-network-mock.md §10): with the hook in
// initialize() the release binary carries ~42KB of this package's rules engine —
// the linker reaches mock.go through ApplyMockFromEnv and cannot drop it. Without
// the hook the engine is dead code and disappears.
//
// The check is a source scan rather than a call: initialize() starts the nine
// availability self-tests, which would outlive the test and, once the mock is
// cleared, talk to the real network.
func TestMockM11StartPathDoesNotReadTheMockEnv(t *testing.T) {
	const path = "tool.go"

	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("M11: reading %s: %v", path, err)
	}

	if strings.Contains(string(src), "ApplyMockFromEnv") {
		t.Fatalf("M11: %s calls ApplyMockFromEnv. That single call makes the linker keep the whole mock rules engine in every release binary (~42KB); the mock belongs to tests and debug builds, which call ApplyMockFromEnv themselves.", path)
	}
}

// M10: concurrent requests, Requests()/Unmatched() reads and
// SetMock/ClearMock churn are race free.
func TestMockM10ConcurrentAccess(t *testing.T) {
	m := NewMockTransport(MockRule{Name: "local", Host: "127.0.0.1", Body: []byte("ok")})
	SetMock(m)
	t.Cleanup(ClearMock)

	stop := make(chan struct{})
	var churn sync.WaitGroup
	churn.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			SetMock(m)
			ClearMock()
		}
	})

	var served atomic.Int64
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Go(func() {
			// one provider per worker: AddRedirectionInterceptor mutates the
			// shared *http.Client and would race with itself otherwise
			p := NewProxyProvider("", "m10", testCase{})
			for j := 0; j < 25; j++ {
				resp, err := doGet(p, deadEndpoint+"/m10", nil)
				if err != nil {
					// the mock may have been cleared concurrently; a request
					// that escaped must fail rather than reach a real server
					continue
				}

				body, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr == nil && resp.StatusCode == http.StatusOK && string(body) == "ok" {
					served.Add(1)
				}

				_ = m.Requests()
				_ = m.Unmatched()
			}
		})
	}

	workers.Wait()
	close(stop)
	churn.Wait()

	if served.Load() == 0 {
		t.Fatalf("M10: no request was served by the mock while SetMock/ClearMock were churning")
	}

	SetMock(m)
	if _, body := mustGet(t, newTestProvider(), deadEndpoint+"/m10", nil); body != "ok" {
		t.Fatalf("M10: after the churn the mock must serve again, got body=%q", body)
	}
	_ = m.Requests()
	_ = m.Unmatched()
}
