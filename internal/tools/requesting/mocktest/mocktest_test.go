package mocktest_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting/mocktest"
)

func get(t *testing.T, rawURL string) (*http.Response, string) {
	t.Helper()

	p := requesting.GetClient(requesting.NoProxy)
	req, err := p.NewGetRequest(rawURL, context.Background())
	if err != nil {
		t.Fatalf("GET %s: building request: %v", rawURL, err)
	}

	resp, err := p.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("GET %s: reading body: %v", rawURL, err)
	}
	return resp, string(body)
}

func TestSetupHelpers(t *testing.T) {
	t.Run("rules", func(t *testing.T) {
		m := mocktest.Setup(t,
			mocktest.Text("text.test", "/t", "hello"),
			mocktest.JSON("json.test", "/j", map[string]string{"tag_name": "v1"}),
			mocktest.Body("body.test", "/b", http.StatusTeapot, []byte("teapot")),
		)

		resp, body := get(t, "http://text.test/t")
		if resp.StatusCode != http.StatusOK || body != "hello" {
			t.Fatalf("mocktest.Text: got status=%d body=%q, want status=200 body=\"hello\"", resp.StatusCode, body)
		}
		if got := resp.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
			t.Fatalf("mocktest.Text: Content-Type = %q, want \"text/plain; charset=utf-8\"", got)
		}

		resp, body = get(t, "http://json.test/j")
		if resp.StatusCode != http.StatusOK || body != `{"tag_name":"v1"}` {
			t.Fatalf("mocktest.JSON: got status=%d body=%q, want status=200 body=%q", resp.StatusCode, body, `{"tag_name":"v1"}`)
		}
		if got := resp.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("mocktest.JSON: Content-Type = %q, want \"application/json\"", got)
		}

		resp, body = get(t, "http://body.test/b")
		if resp.StatusCode != http.StatusTeapot || body != "teapot" {
			t.Fatalf("mocktest.Body: got status=%d body=%q, want status=418 body=\"teapot\"", resp.StatusCode, body)
		}
		if resp.Status != "418 I'm a teapot" {
			t.Fatalf("mocktest.Body: Status = %q, want \"418 I'm a teapot\"", resp.Status)
		}

		if got := len(m.Requests()); got != 3 {
			t.Fatalf("mocktest: the mock served %d requests, want 3", got)
		}
		mocktest.AssertAllMatched(t, m)
	})

	if requesting.Mock() != nil {
		t.Fatalf("mocktest.Setup must clear the mock when the test ends, Mock()=%v", requesting.Mock())
	}
}

// availabilityRule returns the rule MockAvailabilityRules() generates for one
// self-test. The URLs below are rebuilt from it, so this test cannot keep passing
// while the probe targets move (the YouTube API host did move once already).
func availabilityRule(t *testing.T, client requesting.ClientName) requesting.MockRule {
	t.Helper()

	want := "availability:" + string(client)
	for _, rule := range requesting.MockAvailabilityRules() {
		if rule.Name == want {
			return rule
		}
	}

	t.Fatalf("no availability rule named %q: the self-test for %q would go to the real network in the tests below", want, client)
	return requesting.MockRule{}
}

func TestSetupFileAndAvailability(t *testing.T) {
	t.Run("setup file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "mock.yaml")
		if err := os.WriteFile(path, []byte("rules:\n  - name: file\n    host: file.test\n    path: /f\n    body: from-file\n"), 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}

		mocktest.SetupFile(t, path)

		resp, body := get(t, "http://file.test/f")
		if resp.StatusCode != http.StatusOK || body != "from-file" {
			t.Fatalf("mocktest.SetupFile: got status=%d body=%q, want status=200 body=\"from-file\"", resp.StatusCode, body)
		}
	})

	t.Run("availability", func(t *testing.T) {
		m := mocktest.Setup(t, mocktest.Availability()...)

		cases := []struct {
			client requesting.ClientName
			note   string
		}{
			{requesting.GitHubApi, "GitHub API"},
			{requesting.YouTubeApi, "YouTube API"},
			{requesting.GitHubAssets, "GitHub assets"},
			{requesting.PyPyDance, "PyPy video"},
		}

		for _, tc := range cases {
			rule := availabilityRule(t, tc.client)

			resp, _ := get(t, "http://"+rule.Host+rule.Path)
			if resp.StatusCode != rule.Status {
				t.Fatalf("mocktest.Availability: %s self-test got %d, want %d", tc.note, resp.StatusCode, rule.Status)
			}
			if resp.ContentLength != int64(len(rule.Body)) {
				t.Fatalf("mocktest.Availability: %s: ContentLength = %d, want the rule body length %d", tc.note, resp.ContentLength, len(rule.Body))
			}
			if tc.client == requesting.PyPyDance && resp.ContentLength < 1024*1024 {
				t.Fatalf("mocktest.Availability: PyPy video self-test got ContentLength=%d, want at least 1 MiB", resp.ContentLength)
			}
		}

		mocktest.AssertAllMatched(t, m)

		// A URL that no rule covers must fail, and it has to show up as unmatched.
		// A catch-all rule (every availability rule collapsed into one rule that
		// matches anything) would answer it instead, which is exactly why
		// Availability returns the per-case rules: AssertAllMatched would never see
		// a request a test forgot to describe.
		p := requesting.GetClient(requesting.NoProxy)
		req, err := p.NewGetRequest("http://unknown.test/x", context.Background())
		if err != nil {
			t.Fatalf("building request: %v", err)
		}
		if _, err := p.Do(req); err == nil {
			t.Fatal("mocktest.Availability: an unknown URL must fail instead of matching anything silently")
		} else if !strings.Contains(err.Error(), "no mock rule for GET http://unknown.test/x") {
			t.Fatalf("mocktest.Availability: error = %q, want the strict no-rule error", err.Error())
		}

		unmatched := m.Unmatched()
		if len(unmatched) != 1 || !strings.Contains(unmatched[0], "unknown.test/x") {
			t.Fatalf("mocktest.Availability: Unmatched() = %v, want the one request no rule covered", unmatched)
		}
	})
}
