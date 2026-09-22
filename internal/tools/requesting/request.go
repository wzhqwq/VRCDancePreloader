package requesting

import (
	"errors"
	"net/http"

	"google.golang.org/api/option"
)

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"

type mixedTransport struct {
	clientWithProxy *http.Client
	Key             string
}

func (t *mixedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := *req
	args := newReq.URL.Query()
	args.Set("key", t.Key)
	u := *req.URL
	newReq.URL = &u
	newReq.URL.RawQuery = args.Encode()
	return t.clientWithProxy.Transport.RoundTrip(&newReq)
}

// WithYoutubeApiClient builds the option that makes the generated YouTube service
// use the client configured for YouTubeApi (so the request goes through the proxy
// the user configured for it) and injects the API key as a query parameter.
//
// It reports an error instead of dereferencing a missing provider: the provider is
// filled in by initialize(), and every call site runs after that, so finding
// nothing here means the caller asked too early.
func WithYoutubeApiClient(key string) (option.ClientOption, error) {
	provider := clients[YouTubeApi]
	if provider == nil {
		return nil, errors.New("the YouTube API client is not initialized yet")
	}

	return option.WithHTTPClient(&http.Client{
		Transport: &mixedTransport{
			clientWithProxy: provider.client,
			Key:             key,
		},
	}), nil
}

func SetupHeader(req *http.Request, referer string) {
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", defaultUA)
}
