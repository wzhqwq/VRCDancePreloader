package requesting

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
	"google.golang.org/api/option"
	"net/http"
)

func RequestVideo(url string) (resp *http.Response, err error) {
	if _, ok := utils.CheckPyPyUrl(url); ok {
		return pypyClient.Get(url)
	}
	if _, ok := utils.CheckYoutubeURL(url); ok {
		return youtubeVideoClient.Get(url)
	}
	return http.Get(url)
}

type mixedTransport struct {
	clientWithProxy *http.Client
	Key             string
}

func (t *mixedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := *req
	args := newReq.URL.Query()
	args.Set("key", t.Key)
	newReq.URL.RawQuery = args.Encode()
	return t.clientWithProxy.Transport.RoundTrip(&newReq)
}

func WithYoutubeApiClient(key string) option.ClientOption {
	return option.WithHTTPClient(&http.Client{
		Transport: &mixedTransport{
			clientWithProxy: youtubeApiClient,
			Key:             key,
		},
	})
}
