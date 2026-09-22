// Package mocktest provides small conveniences for tests that run against the
// requesting package's HTTP mock. It lives in its own package because the
// production package must not import "testing".
package mocktest

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
)

// Setup installs a mock built from rules and clears it when the test finishes.
// It fails the test when a mock is already installed, so that nested or
// back-to-back tests cannot silently inherit each other's rules.
func Setup(t testing.TB, rules ...requesting.MockRule) *requesting.MockTransport {
	t.Helper()

	m := requesting.NewMockTransport(rules...)
	install(t, m)
	return m
}

// SetupFile installs the mock described by a rule file (see
// requesting.LoadMockRules; a relative bodyFile is resolved against the rule
// file's directory) and clears it when the test finishes.
func SetupFile(t testing.TB, path string) *requesting.MockTransport {
	t.Helper()

	m, err := requesting.LoadMockRules(path)
	if err != nil {
		t.Fatalf("mocktest.SetupFile: %v", err)
	}

	install(t, m)
	return m
}

func install(t testing.TB, m *requesting.MockTransport) {
	t.Helper()

	if existing := requesting.Mock(); existing != nil {
		t.Fatalf("mocktest: a mock (%T) is already installed; clear it (requesting.ClearMock) before installing another one", existing)
	}

	requesting.SetMock(m)
	t.Cleanup(requesting.ClearMock)
}

// Availability returns the rules that satisfy every availability self-test
// started by the requesting package (requesting.MockAvailabilityRules, one rule
// per test case). Append whatever else a test needs after them:
//
//	mocktest.Setup(t, append(mocktest.Availability(), myRules...)...)
//
// It deliberately does *not* return one catch-all rule: a rule that matches any
// request would answer URLs no rule covers, so the request would never be
// recorded as unmatched and AssertAllMatched (the check that a test did not
// forget a rule) would stop working.
func Availability() []requesting.MockRule {
	return requesting.MockAvailabilityRules()
}

// JSON returns a rule that answers host+path with 200 and v as JSON.
func JSON(host, path string, v any) requesting.MockRule {
	return requesting.MockRule{
		Name:     "json " + host + path,
		Host:     host,
		Path:     path,
		Status:   http.StatusOK,
		BodyJSON: v,
	}
}

// Body returns a rule that answers host+path with the given status and payload.
func Body(host, path string, status int, body []byte) requesting.MockRule {
	return requesting.MockRule{
		Name:   fmt.Sprintf("body %s%s (%d)", host, path, status),
		Host:   host,
		Path:   path,
		Status: status,
		Body:   body,
	}
}

// Text returns a rule that answers host+path with 200 and a plain text body.
func Text(host, path, body string) requesting.MockRule {
	return requesting.MockRule{
		Name:   "text " + host + path,
		Host:   host,
		Path:   path,
		Status: http.StatusOK,
		Header: http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
		Body:   []byte(body),
	}
}

// AssertAllMatched reports every request that reached the mock without matching
// a rule, which usually means the test forgot a rule or uses a different URL
// than expected.
func AssertAllMatched(t testing.TB, m *requesting.MockTransport) {
	t.Helper()

	if unmatched := m.Unmatched(); len(unmatched) > 0 {
		t.Errorf("mocktest: %d request(s) matched no mock rule: %v", len(unmatched), unmatched)
	}
}
