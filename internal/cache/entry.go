package cache

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Entry interface {
	io.Writer
	Open() error
	TotalLen() int64
	GetReadSeekCloser() io.ReadSeekCloser
	GetDownloadBody() (io.ReadCloser, error)
	Close() error
	Save() error
	IsComplete() bool
}

type BaseEntry struct {
	id     string
	client *http.Client

	writingFile *ReadableWritingFile
}

func (e *BaseEntry) getSavedVideoName() string {
	return fmt.Sprintf("%s/%s.mp4", cachePath, e.id)
}
func (e *BaseEntry) getIncompleteVideoName() string {
	return fmt.Sprintf("%s/%s.mp4.dl", cachePath, e.id)
}
func (e *BaseEntry) getSavedSize() int64 {
	return getFileSize(e.getSavedVideoName())
}
func (e *BaseEntry) getIncompleteSize() int64 {
	return getFileSize(e.getIncompleteVideoName())
}
func (e *BaseEntry) openFile() error {
	if e.writingFile == nil {
		if e.getSavedSize() > 0 {
			e.writingFile = newReadableWritingFile(e.getSavedVideoName())
		} else {
			e.writingFile = newReadableWritingFile(e.getIncompleteVideoName())
		}
		if e.writingFile == nil {
			return fmt.Errorf("cannot open or create video file")
		}
	}
	return nil
}
func (e *BaseEntry) saveFile() error {
	if e.writingFile != nil && strings.Contains(e.writingFile.file.Name(), ".dl") {
		return e.writingFile.Rename(e.getSavedVideoName())
	}
	return nil
}
func (e *BaseEntry) closeFile() error {
	if e.writingFile != nil {
		return e.writingFile.Close()
	}
	return nil
}

func (e *BaseEntry) requestInfo(url string) (int64, string) {
	res, err := e.client.Head(url)
	if err != nil {
		return 0, url
	}
	location := res.Header.Get("Location")
	if location == "" {
		return res.ContentLength, url
	}
	return res.ContentLength, location
}
func (e *BaseEntry) requestBody(url string, offset int64) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	res, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}

	if offset == 0 || res.StatusCode == http.StatusPartialContent {
		return res.Body, nil
	}
	return nil, fmt.Errorf(res.Status)
}
