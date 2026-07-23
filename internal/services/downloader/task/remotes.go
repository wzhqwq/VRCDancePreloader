package task

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type rangedRemoteProvider struct {
	url     string
	referer string
	client  *requesting.ClientProvider
}

func (p *rangedRemoteProvider) WaitResolving(_ context.Context, _ func()) (int64, error) {
	return 0, nil
}

func (p *rangedRemoteProvider) GetDownloadStream(offset int64, ctx context.Context) (StreamInfo, error) {
	logger.InfoLn("Request body", p.url, offset)
	req, err := p.client.NewBoundGetRequest(p.url, ctx)
	if err != nil {
		return StreamInfo{}, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	requesting.SetupHeader(req, p.referer)
	res, err := p.client.Do(req)
	if err != nil {
		return StreamInfo{}, err
	}

	if res.StatusCode >= 500 {
		return StreamInfo{}, fmt.Errorf("%w%s is temporarily unavailable: %d", interactive.ErrTemporarilyUnavailable, p.url, res.StatusCode)
	}
	if res.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return StreamInfo{}, nil
	}
	if res.StatusCode == http.StatusOK {
		if offset > 0 {
			logger.WarnLn(p.url, "doesn't support the 'Range' header, so we will download it from the start")
		}
		return StreamInfo{res.Body, res.ContentLength, false}, nil
	}
	if res.StatusCode == http.StatusPartialContent {
		return StreamInfo{res.Body, offset + res.ContentLength, true}, nil
	}
	if res.StatusCode == http.StatusTooManyRequests {
		retryAfter, _ := strconv.ParseInt(res.Header.Get("Retry-After"), 10, 32)
		return StreamInfo{}, utils.NewThrottledError(time.Duration(retryAfter) * time.Second)
	}

	return StreamInfo{}, fmt.Errorf("unexpected status code: %d", res.StatusCode)
}

var _ RemoteProvider = &rangedRemoteProvider{}

func NewRangedRemoteProvider(url string, client *requesting.ClientProvider) RemoteProvider {
	return &rangedRemoteProvider{url, url, client}
}

type HandleFn func(id string) *interactive.RemoteHandle[*types.RemoteHttpResourceInfo]

type rwFileRemoteProvider struct {
	rangedRemoteProvider

	id string

	session types.CDNFileSession

	handleFn HandleFn
}

func (p *rwFileRemoteProvider) tryResolveFromCache() (int64, bool) {
	file, err := p.session.AcquireFile()
	if err != nil {
		p.session.Logger().WarnLn("Failed to inspect local cache, falling back to remote resolution:", err)
		return 0, false
	}
	defer p.session.ReleaseFile()

	if file.IsComplete() && !p.session.IsForceExpiration() {
		return file.TotalLen(), true
	}

	return 0, false
}

func (p *rwFileRemoteProvider) WaitResolving(ctx context.Context, beforeWait func()) (int64, error) {
	if l, ok := p.tryResolveFromCache(); ok {
		return l, nil
	}

	handle := p.handleFn(p.id)
	defer handle.Release()

	if !handle.Snapshot().Status.Valid() {
		beforeWait()
	}
	info, err := handle.BlockedGet(ctx)
	if err != nil {
		return 0, err
	}

	p.session.ReconcileRemoteInfo(info)

	p.url = info.FinalUrl
	p.referer = info.Referer
	if info.Client != nil {
		p.client = info.Client
	}

	p.session.Logger().InfoLn(p.id, "resolved to", info.FinalUrl, "size:", info.TotalSize, "modified time:", info.LastModified.Local().String(), "etag:", info.Etag)

	return info.TotalSize, nil
}

func NewRWFileRemoteProvider(id string, handleFn HandleFn, session types.CDNFileSession) RemoteProvider {
	return &rwFileRemoteProvider{
		id:       id,
		session:  session,
		handleFn: handleFn,
	}
}
