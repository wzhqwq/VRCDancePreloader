package cache

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type UrlBasedEntry struct {
	BaseEntry

	resolvedUrl      string
	initialUrlGetter func(ctx context.Context) (string, error)
}

func newUrlBasedEntry(id string, client *http.Client, initialUrlGetter func(ctx context.Context) (string, error)) *UrlBasedEntry {
	return &UrlBasedEntry{
		BaseEntry:        ConstructBaseEntry(id, client),
		initialUrlGetter: initialUrlGetter,
	}
}

func (e *UrlBasedEntry) resolveUrl(ctx context.Context) error {
	url, err := e.initialUrlGetter(ctx)
	if err != nil {
		return err
	}

	var info *RemoteVideoInfo

	for {
		if _, ok := utils.CheckYoutubeURL(url); ok {
			// TODO youtube handler
			return ErrNotSupported
		}
		info, err = e.requestHttpResInfo(url, ctx)
		if err != nil {
			return err
		}
		changed := info.FinalUrl != url
		url = info.FinalUrl

		// check if it's a real video, at least 1MB
		if info.TotalSize > 1024*1024 {
			break
		} else if !changed {
			// not redirected and not a video
			return ErrNotSupported
		}
	}

	e.workingFile.Init(info.TotalSize, info.LastModified)

	e.resolvedUrl = url
	return nil
}

func (e *UrlBasedEntry) TotalLen() (int64, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile.TotalLen() == 0 {
		err := e.resolveUrl(context.Background())
		if err != nil {
			return 0, err
		}
	}

	return e.workingFile.TotalLen(), nil
}
func (e *UrlBasedEntry) GetDownloadStream() (io.ReadCloser, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.resolvedUrl == "" {
		err := e.resolveUrl(context.Background())
		if err != nil {
			return nil, err
		}
	}

	e.workingFile.MarkDownloading()
	offset := e.workingFile.GetDownloadOffset()

	log.Printf("Download %s start from %d, (total %d)", e.id, offset, e.workingFile.TotalLen())

	return e.requestHttpResBody(e.resolvedUrl, offset, context.Background())
}
func (e *UrlBasedEntry) Reset() {
	e.resolvedUrl = ""
}
func (e *UrlBasedEntry) GetReadSeeker(ctx context.Context) (io.ReadSeeker, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return nil, io.ErrClosedPipe
	}

	if e.resolvedUrl == "" {
		err := e.resolveUrl(ctx)
		if err != nil {
			return nil, err
		}
	}

	return e.getReadSeeker(ctx)
}
