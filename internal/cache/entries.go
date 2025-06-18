package cache

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
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
	// TODO uncomment when WannaDance queue processing is well tested
	if num, ok := utils.CheckIdIsWanna(id); ok {
		return &DirectDownloadEntry{
			BaseEntry: BaseEntry{
				id:     id,
				client: requesting.GetPyPyClient(),
			},
			videoUrl: utils.GetWannaVideoUrl(num),
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

func (p *DirectDownloadEntry) Open() error {
	return p.openFile()
}

func (p *DirectDownloadEntry) TotalLen() int64 {
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

func (p *DirectDownloadEntry) GetReadSeekCloser() io.ReadSeekCloser {
	if totalLen := p.TotalLen(); totalLen > 0 {
		return p.writingFile.RequestRsc(totalLen)
	}
	return nil
}

func (p *DirectDownloadEntry) GetDownloadBody() (io.ReadCloser, error) {
	return p.requestBody(p.videoUrl, p.getIncompleteSize())
}

func (p *DirectDownloadEntry) Write(bytes []byte) (int, error) {
	err := p.writingFile.Append(bytes)
	if err != nil {
		return 0, err
	}
	return len(bytes), nil
}

func (p *DirectDownloadEntry) Close() error {
	return p.closeFile()
}

func (p *DirectDownloadEntry) Save() error {
	return p.saveFile()
}

func (p *DirectDownloadEntry) IsComplete() bool {
	return p.getSavedSize() > 0
}

type YouTubeEntry struct {
	BaseEntry
}
