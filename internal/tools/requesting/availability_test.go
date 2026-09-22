package requesting

import (
	"context"
	"net/url"
	"testing"

	"google.golang.org/api/youtube/v3"
)

// A1: the availability self-test probes the endpoint the API client really calls.
//
// The probe used to point at www.googleapis.com while the generated
// google.golang.org/api youtube/v3 service talks to youtube.googleapis.com
// (basePath in youtube-gen.go). A self-test on a host the API never uses measures
// the wrong thing: it can report the service as reachable while every real API
// call fails.
//
// The expectation is taken from the library itself (svc.BasePath), not from a
// host written down here, so a future version that moves its endpoint moves the
// probe with it.
func TestAvailabilityProbeTargetsTheRealEndpoint(t *testing.T) {
	probe := testCases[YouTubeApi]
	if probe.url == "" {
		t.Fatal("the YouTubeApi test case has no URL to probe")
	}
	probeURL, err := url.Parse(probe.url)
	if err != nil {
		t.Fatalf("parsing the probe URL %q: %v", probe.url, err)
	}

	// WithYoutubeApiClient reads clients[YouTubeApi] (request.go), so the provider
	// has to exist. doTest=false: no availability goroutine is wanted here.
	initClient(YouTubeApi, "", false)

	svc, err := youtube.NewService(context.Background(), WithYoutubeApiClient("test-api-key"))
	if err != nil {
		t.Fatalf("building the youtube service: %v", err)
	}
	apiURL, err := url.Parse(svc.BasePath)
	if err != nil {
		t.Fatalf("parsing the service base path %q: %v", svc.BasePath, err)
	}

	if probeURL.Hostname() != apiURL.Hostname() {
		t.Fatalf("the YouTubeApi self-test probes %q but the API client calls %q: the self-test would measure a host the API never uses", probeURL.Hostname(), apiURL.Hostname())
	}
}

// A2: the failure message follows the configuration, not the client's Transport.
//
// The Transport is never nil any more (see gate), so the old `client.Transport ==
// nil` test had stopped distinguishing anything: every failure was reported as
// "through provided proxy", even for a client with no proxy configured.
func TestProxyHintFollowsTheConfiguration(t *testing.T) {
	if got := proxyHintSuffix(""); got != ", maybe you should configure proxy" {
		t.Fatalf("proxyHintSuffix(\"\") = %q, want the \"configure a proxy\" hint", got)
	}
	if got := proxyHintSuffix("http://127.0.0.1:3128"); got != " through provided proxy" {
		t.Fatalf("proxyHintSuffix(proxy) = %q, want the \"through provided proxy\" wording", got)
	}
}
