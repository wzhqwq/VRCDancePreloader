package cache

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"io"
	"log"
)

func NewEntry(id string) Entry {
	if num, ok := utils.CheckIdIsPyPy(id); ok {
		return &DirectDownloadEntry{
			BaseEntry: ConstructBaseEntry(id, requesting.GetPyPyClient()),
			videoUrl:  utils.GetPyPyVideoUrl(num),
		}
	}
	if num, ok := utils.CheckIdIsWanna(id); ok {
		return &DirectDownloadEntry{
			BaseEntry: ConstructBaseEntry(id, requesting.GetWannaClient()),
			videoUrl:  utils.GetWannaVideoUrl(num),
		}
	}
	if bvID, ok := utils.CheckIdIsBili(id); ok {
		return &BiliBiliEntry{
			DirectDownloadEntry: DirectDownloadEntry{
				BaseEntry: ConstructBaseEntry(id, requesting.GetBiliClient()),
			},
			bvID: bvID,
		}
	}
	// TODO youtube handler
	return nil
}

type DirectDownloadEntry struct {
	BaseEntry
	videoUrl string
}

func (e *DirectDownloadEntry) getTotalLen() int64 {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return 0
	}

	return e.workingFile.GetTotalLength(func() int64 {
		totalLen, newUrl := e.requestHttpResInfo(e.videoUrl)
		e.videoUrl = newUrl
		return totalLen
	})
}

func (e *DirectDownloadEntry) getDownloadStream() (io.ReadCloser, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return nil, io.ErrClosedPipe
	}
	e.workingFile.MarkDownloading()
	offset := e.workingFile.GetDownloadOffset()

	log.Printf("Download %s start from %d", e.id, offset)

	return e.requestHttpResBody(e.videoUrl, offset)
}

// adapters

func (e *DirectDownloadEntry) TotalLen() int64 {
	return e.getTotalLen()
}

func (e *DirectDownloadEntry) GetDownloadStream() (io.ReadCloser, error) {
	return e.getDownloadStream()
}

type BiliBiliEntry struct {
	DirectDownloadEntry

	bvID string
}

func (e *BiliBiliEntry) TotalLen() int64 {
	if e.videoUrl == "" {
		e.videoUrl, _ = third_party_api.GetBiliVideoUrl(e.client, e.bvID)
		if e.videoUrl == "" {
			return 0
		}
	}
	return e.getTotalLen()
}

func (e *BiliBiliEntry) GetDownloadStream() (io.ReadCloser, error) {
	if e.videoUrl == "" {
		var err error
		e.videoUrl, err = third_party_api.GetBiliVideoUrl(e.client, e.bvID)
		if err != nil {
			return nil, err
		}
	}
	return e.getDownloadStream()
}

type YouTubeEntry struct {
	BaseEntry
}
