package third_parties

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type ErrRefused struct {
	Redirection string
}

func (e *ErrRefused) Error() string {
	return "the remote server refuse to provide a video"
}

var urlResolverLogger = utils.NewLogger("URL Resolver")

func directResolve(url string, client *requesting.ClientProvider, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	urlResolverLogger.InfoLn("Resolving", url, "directly")
	req, err := client.NewGetRequest(url, ctx)
	if err != nil {
		return nil, err
	}

	requesting.SetupHeader(req, url)
	//if e.etag != "" {
	//	req.Header.Set("If-None-Match", e.etag)
	//}
	res, err := client.Do(req)
	if err != nil {
		urlResolverLogger.ErrorLn("Failed to get ", url, "reason:", err)
		return nil, fmt.Errorf("failed to get %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		retryAfter, _ := strconv.ParseInt(res.Header.Get("Retry-After"), 10, 32)
		return nil, utils.NewThrottledError(time.Duration(retryAfter) * time.Second)
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusFound || res.StatusCode == http.StatusMovedPermanently {
			// it's intercepted YouTube request
			return nil, &ErrRefused{Redirection: res.Header.Get("Location")}
		}
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var lastModified time.Time
	if lastModifiedText := res.Header.Get("Last-Modified"); lastModifiedText != "" {
		lastModified, _ = http.ParseTime(lastModifiedText)
	}

	return &types.RemoteHttpResourceInfo{
		FinalUrl:     res.Request.URL.String(),
		TotalSize:    res.ContentLength,
		LastModified: lastModified,
		Etag:         res.Header.Get("ETag"),
		Referer:      res.Request.Header.Get("Referer"),

		Client: client,
	}, nil
}

func nopResolve(url string, client *requesting.ClientProvider) (*types.RemoteHttpResourceInfo, error) {
	return &types.RemoteHttpResourceInfo{
		FinalUrl: url,

		Client: client,
	}, nil
}

func ytDlpResolve(url string, client *requesting.ClientProvider, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if !local_executables.YtDlpAvailable() {
		return nil, ErrYtDlpNotAvailable
	}
	urlResolverLogger.InfoLn("Resolving", url, "using yt-dlp")

	url, err := local_executables.ResolveVideoUrlWithYtDlp(url, ctx)
	if err != nil {
		return nil, err
	}

	return directResolve(url, client, ctx)
}
