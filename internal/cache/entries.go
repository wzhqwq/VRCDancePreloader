package cache

import (
	"context"
	"io"
	"log"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
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

func (e *DirectDownloadEntry) init(ctx context.Context) error {
	if e.workingFile == nil {
		return io.ErrClosedPipe
	}
	if e.workingFile.TotalLen() > 0 {
		return nil
	}

	info, err := e.requestHttpResInfo(e.videoUrl, ctx)
	if err != nil {
		return err
	}

	e.videoUrl = info.FinalUrl
	e.workingFile.Init(info.TotalSize, info.LastModified)

	return nil
}

func (e *DirectDownloadEntry) getTotalLen() (int64, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	err := e.init(context.Background())
	if err != nil {
		return 0, err
	}

	return e.workingFile.TotalLen(), nil
}

func (e *DirectDownloadEntry) getDownloadStream() (io.ReadCloser, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	e.workingFile.MarkDownloading()
	offset := e.workingFile.GetDownloadOffset()

	log.Printf("Download %s start from %d", e.id, offset)

	return e.requestHttpResBody(e.videoUrl, offset, context.Background())
}

func (e *DirectDownloadEntry) getReadSeekerWithInit(ctx context.Context) (io.ReadSeeker, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return nil, io.ErrClosedPipe
	}

	err := e.init(ctx)
	if err != nil {
		return nil, err
	}

	return e.getReadSeeker(ctx)
}

// adapters

func (e *DirectDownloadEntry) TotalLen() (int64, error) {
	return e.getTotalLen()
}
func (e *DirectDownloadEntry) GetDownloadStream() (io.ReadCloser, error) {
	return e.getDownloadStream()
}
func (e *DirectDownloadEntry) GetReadSeeker(ctx context.Context) (io.ReadSeeker, error) {
	return e.getReadSeekerWithInit(ctx)
}

type BiliBiliEntry struct {
	DirectDownloadEntry

	bvID string
}

func (e *BiliBiliEntry) TotalLen() (int64, error) {
	if e.videoUrl == "" {
		var err error
		e.videoUrl, err = third_party_api.GetBiliVideoUrl(e.client, e.bvID, context.Background())
		if err != nil {
			return 0, err
		}
	}
	return e.getTotalLen()
}

func (e *BiliBiliEntry) GetDownloadStream() (io.ReadCloser, error) {
	if e.videoUrl == "" {
		var err error
		e.videoUrl, err = third_party_api.GetBiliVideoUrl(e.client, e.bvID, context.Background())
		if err != nil {
			return nil, err
		}
	}
	return e.getDownloadStream()
}

func (e *BiliBiliEntry) GetReadSeeker(ctx context.Context) (io.ReadSeeker, error) {
	if e.videoUrl == "" {
		var err error
		e.videoUrl, err = third_party_api.GetBiliVideoUrl(e.client, e.bvID, ctx)
		if err != nil {
			return nil, err
		}
	}
	return e.getReadSeekerWithInit(ctx)
}

type YouTubeEntry struct {
	BaseEntry
}
