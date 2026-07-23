package types

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
)

type GeneralVideoInfo struct {
	Title     string
	Duration  time.Duration
	GroupName string

	FallbackUrl string
}

func (i GeneralVideoInfo) CompleteIfEmpty(title, group string, duration int) GeneralVideoInfo {
	if i.Duration == 0 {
		i.Title = title
		i.Duration = time.Duration(duration) * time.Second
		i.GroupName = group
	}
	return i
}

type RemoteHttpResourceInfo struct {
	FinalUrl     string
	TotalSize    int64
	LastModified time.Time
	Etag         string
	Referer      string

	Client *requesting.ClientProvider
}
