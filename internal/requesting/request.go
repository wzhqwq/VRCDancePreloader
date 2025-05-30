package requesting

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"google.golang.org/api/option"
	"net/http"
)

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

func RequestThumbnail(url string) (resp *http.Response, err error) {
	if utils.CheckPyPyThumbnailUrl(url) {
		return pypyClient.Get(url)
	}
	if _, ok := utils.CheckYoutubeURL(url); ok {
		return youtubeImageClient.Get(url)
	}
	return http.Get(url)
}
