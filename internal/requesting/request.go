package requesting

import (
	"context"
	"net/http"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"golang.org/x/sync/semaphore"
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
	newReq.URL.RawQuery = args.Encode()
	return t.clientWithProxy.Transport.RoundTrip(&newReq)
}

func WithYoutubeApiClient(key string) option.ClientOption {
	return option.WithHTTPClient(&http.Client{
		Transport: &mixedTransport{
			clientWithProxy: clients[YouTubeApi].client,
			Key:             key,
		},
	})
}

var thumbnailRequestSem = semaphore.NewWeighted(6)

func RequestThumbnail(url string) (*http.Response, error) {
	err := thumbnailRequestSem.Acquire(context.Background(), 1)
	if err != nil {
		return nil, err
	}
	defer thumbnailRequestSem.Release(1)

	if utils.CheckPyPyResource(url) {
		return clients[PyPyDance].Get(url)
	}
	if utils.CheckYoutubeThumbnailURL(url) {
		return clients[YouTubeImage].Get(url)
	}
	return http.Get(url)
}

func SetupHeader(req *http.Request, referer string) {
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", defaultUA)
}
