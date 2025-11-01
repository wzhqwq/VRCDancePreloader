package requesting

import (
	"net/http"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"google.golang.org/api/option"
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
	if utils.CheckPyPyResource(url) {
		return pypyClient.Get(url)
	}
	if utils.CheckYoutubeThumbnailURL(url) {
		return youtubeImageClient.Get(url)
	}
	return http.Get(url)
}
