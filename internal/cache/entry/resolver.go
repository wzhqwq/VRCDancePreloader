package entry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type RemoteVideoInfo struct {
	FinalUrl     string
	TotalSize    int64
	LastModified time.Time
	Etag         string
	Referer      string

	Client *requesting.ClientProvider
}

type UrlResolver interface {
	Resolve(logger utils.LoggerImpl, ctx context.Context) (*RemoteVideoInfo, error)
	Next(hintUrl string) UrlResolver
	Check(hintUrl string) bool
}

type BaseUrlResolver struct {
	Fallbacks []UrlResolver
}

func (r *BaseUrlResolver) Next(hintUrl string) UrlResolver {
	for _, fallback := range r.Fallbacks {
		if fallback.Check(hintUrl) {
			return fallback
		}
	}
	return nil
}

func (r *BaseUrlResolver) Check(string) bool {
	return false
}

func ConstructBaseUrlResolver(fallbacks ...UrlResolver) BaseUrlResolver {
	return BaseUrlResolver{
		Fallbacks: fallbacks,
	}
}

type DirectUrlResolver struct {
	BaseUrlResolver

	url    string
	client *requesting.ClientProvider
}

func (r *DirectUrlResolver) Resolve(logger utils.LoggerImpl, ctx context.Context) (*RemoteVideoInfo, error) {
	logger.InfoLn("Request info", r.url)
	req, err := r.client.NewGetRequest(r.url, ctx)
	if err != nil {
		return nil, err
	}

	requesting.SetupHeader(req, r.url)
	//if e.etag != "" {
	//	req.Header.Set("If-None-Match", e.etag)
	//}
	res, err := r.client.Do(req)
	if err != nil {
		logger.ErrorLn("Failed to get ", r.url, "reason:", err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrThrottle
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusFound || res.StatusCode == http.StatusMovedPermanently {
			// it's intercepted YouTube request
			return &RemoteVideoInfo{
				FinalUrl: res.Header.Get("Location"),
			}, nil
		}
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	lastModified := unixEpochTime
	if lastModifiedText := res.Header.Get("Last-Modified"); lastModifiedText != "" {
		lastModified, _ = http.ParseTime(lastModifiedText)
	}

	return &RemoteVideoInfo{
		FinalUrl:     res.Request.URL.String(),
		TotalSize:    res.ContentLength,
		LastModified: lastModified,
		Etag:         res.Header.Get("ETag"),
		Referer:      res.Request.Header.Get("Referer"),

		Client: r.client,
	}, nil
}

func (r *DirectUrlResolver) SetUrl(url string) *DirectUrlResolver {
	r.url = url
	return r
}

func NewDirectUrlResolver(url string, client *requesting.ClientProvider, fallbacks ...UrlResolver) *DirectUrlResolver {
	return &DirectUrlResolver{
		BaseUrlResolver: ConstructBaseUrlResolver(fallbacks...),

		url:    url,
		client: client,
	}
}

func ConstructDirectUrlResolver(url string, client *requesting.ClientProvider, fallbacks ...UrlResolver) DirectUrlResolver {
	return DirectUrlResolver{
		BaseUrlResolver: ConstructBaseUrlResolver(fallbacks...),

		url:    url,
		client: client,
	}
}
