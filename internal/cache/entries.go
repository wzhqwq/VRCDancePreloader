package cache

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"io"
)

func NewEntry(id string) Entry {
	if num, ok := utils.CheckIdIsPyPy(id); ok {
		return &DirectDownloadEntry{
			BaseEntry: BaseEntry{
				id:     id,
				client: requesting.GetPyPyClient(),
			},
			videoUrl: utils.GetPyPyVideoUrl(num),
		}
	}
	if num, ok := utils.CheckIdIsWanna(id); ok {
		return &DirectDownloadEntry{
			BaseEntry: BaseEntry{
				id:     id,
				client: requesting.GetWannaClient(),
			},
			videoUrl: utils.GetWannaVideoUrl(num),
		}
	}
	if bvID, ok := utils.CheckIdIsBili(id); ok {
		return &BiliBiliEntry{
			BaseEntry: BaseEntry{
				id:     id,
				client: requesting.GetBiliClient(),
			},
			bvID: bvID,
		}
	}
	// TODO youtube handler
	return nil
}

func OpenEntry(id string) Entry {
	e := NewEntry(id)
	if e == nil {
		return nil
	}

	err := e.Open()
	if err != nil {
		panic(err)
	}
	return e
}

type DirectDownloadEntry struct {
	BaseEntry
	videoUrl string

	totalLen int64
}

func (e *DirectDownloadEntry) TotalLen() int64 {
	if e.totalLen == 0 {
		if savedSize := e.getSavedSize(); savedSize > 0 {
			e.totalLen = savedSize
		} else {
			totalLen, newUrl := e.requestInfo(e.videoUrl)
			e.videoUrl = newUrl
			e.totalLen = totalLen
		}
	}
	return e.totalLen
}

func (e *DirectDownloadEntry) GetReadSeekCloser() io.ReadSeekCloser {
	if totalLen := e.TotalLen(); totalLen > 0 {
		return e.writingFile.RequestRsc(totalLen)
	}
	return nil
}

func (e *DirectDownloadEntry) GetDownloadBody() (io.ReadCloser, error) {
	return e.requestBody(e.videoUrl, e.getIncompleteSize())
}

type BiliBiliEntry struct {
	BaseEntry

	bvID     string
	videoUrl string

	totalLen int64
}

func (e *BiliBiliEntry) TotalLen() int64 {
	if e.totalLen == 0 {
		if savedSize := e.getSavedSize(); savedSize > 0 {
			e.totalLen = savedSize
		} else {
			if e.videoUrl == "" {
				e.videoUrl, _ = third_party_api.GetBiliVideoUrl(e.client, e.bvID)
				if e.videoUrl == "" {
					return 0
				}
			}
			totalLen, newUrl := e.requestInfo(e.videoUrl)
			e.videoUrl = newUrl
			e.totalLen = totalLen
		}
	}
	return e.totalLen
}

func (e *BiliBiliEntry) GetReadSeekCloser() io.ReadSeekCloser {
	if totalLen := e.TotalLen(); totalLen > 0 {
		return e.writingFile.RequestRsc(totalLen)
	}
	return nil
}

func (e *BiliBiliEntry) GetDownloadBody() (io.ReadCloser, error) {
	if e.videoUrl == "" {
		var err error
		e.videoUrl, err = third_party_api.GetBiliVideoUrl(e.client, e.bvID)
		if err != nil {
			return nil, err
		}
	}
	return e.requestBody(e.videoUrl, e.getIncompleteSize())
}

type YouTubeEntry struct {
	BaseEntry
}
