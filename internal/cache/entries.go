package cache

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/requesting"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
	"io"
	"strconv"
	"strings"
)

func OpenEntry(id string) Entry {
	var e Entry
	if strings.Contains(id, "pypy_") {
		num, err := strconv.Atoi(strings.Split(id, "pypy_")[1])
		if err != nil {
			panic(err)
		}
		e = &PyPyEntry{
			BaseEntry: BaseEntry{
				id:     id,
				client: requesting.GetPyPyClient(),
			},
			videoUrl: utils.GetPyPyVideoUrl(num),
		}
	}
	// TODO youtube handler

	if e != nil {
		err := e.Open()
		if err != nil {
			panic(err)
		}
	}
	return e
}

type PyPyEntry struct {
	BaseEntry
	videoUrl string

	totalLen int64
}

func (p *PyPyEntry) Open() error {
	return p.openFile()
}

func (p *PyPyEntry) TotalLen() int64 {
	if p.totalLen == 0 {
		if savedSize := p.getSavedSize(); savedSize != 0 {
			p.totalLen = savedSize
		} else {
			totalLen, newUrl := p.requestInfo(p.videoUrl)
			p.videoUrl = newUrl
			p.totalLen = totalLen
		}
	}
	return p.totalLen
}

func (p *PyPyEntry) GetReadSeekCloser() io.ReadSeekCloser {
	if totalLen := p.TotalLen(); totalLen > 0 {
		return p.writingFile.RequestRsc(totalLen)
	}
	return nil
}

func (p *PyPyEntry) GetDownloadBody() (io.ReadCloser, error) {
	return p.requestBody(p.videoUrl, p.getIncompleteSize())
}

func (p *PyPyEntry) Write(bytes []byte) (int, error) {
	err := p.writingFile.Append(bytes)
	if err != nil {
		return 0, err
	}
	return len(bytes), nil
}

func (p *PyPyEntry) Close() error {
	return p.closeFile()
}

func (p *PyPyEntry) Save() error {
	return p.saveFile()
}

func (p *PyPyEntry) IsComplete() bool {
	return p.getSavedSize() > 0
}

type YouTubeEntry struct {
	BaseEntry
}
