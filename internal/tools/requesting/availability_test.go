package requesting

import (
	"testing"
)

// A3: WithYoutubeApiClient reports a missing provider instead of dereferencing it.
//
// The provider is filled in by initialize() and every call site runs after that, so
// a missing one means the caller asked too early — an error it can log and return
// beats the nil dereference this call used to do (api/youtube.go reaches it on a
// normal code path).
func TestWithYoutubeApiClientReportsAMissingProvider(t *testing.T) {
	previous := clients[YouTubeApi]
	clients[YouTubeApi] = nil
	t.Cleanup(func() { clients[YouTubeApi] = previous })

	clientOption, err := WithYoutubeApiClient("test-api-key")
	if err == nil {
		t.Fatalf("WithYoutubeApiClient returned %v and no error for a missing provider", clientOption)
	}
	if clientOption != nil {
		t.Fatalf("WithYoutubeApiClient returned an option (%v) together with the error", clientOption)
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
