package task

import (
	"io"
	"os"

	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type simpleLocalFileProvider struct {
	downloadedSize int64

	file *os.File
}

func (p *simpleLocalFileProvider) SeekStart() error {
	if p.downloadedSize > 0 {
		_, err := p.file.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		p.downloadedSize = 0
	}
	return nil
}

func (p *simpleLocalFileProvider) CurrentCursor() (int64, error) {
	return p.downloadedSize, nil
}

func (p *simpleLocalFileProvider) Write(data []byte) (n int, err error) {
	n, err = p.file.Write(data)
	if err != nil {
		return
	}
	p.downloadedSize += int64(n)
	return
}

func (p *simpleLocalFileProvider) Open() error {
	// try seek to get downloaded size
	offset, err := p.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	p.downloadedSize = offset
	_, err = p.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	return nil
}

func (p *simpleLocalFileProvider) Close() {
}

func (p *simpleLocalFileProvider) IsComplete() bool {
	// let the downloader decide
	return false
}

func (p *simpleLocalFileProvider) IsForceResolving() bool {
	return false
}

func (p *simpleLocalFileProvider) DownloadedSize() int64 {
	return p.downloadedSize
}

type RemoteLocalProvider interface {
	RemoteProvider
	LocalProvider
}

var _ LocalProvider = &simpleLocalFileProvider{}

func NewLocalFileProvider(file *os.File) LocalProvider {
	return &simpleLocalFileProvider{0, file}
}

type rwFileProvider struct {
	id      string
	session types.CDNFileSession
	logger  utils.LoggerImpl
}

func (p *rwFileProvider) Write(data []byte) (n int, err error) {
	file, err := p.session.AcquireFile()
	if err != nil {
		return 0, err
	}
	defer p.session.ReleaseFile()

	return file.Write(data)
}

func (p *rwFileProvider) Open() error {
	return p.session.Open(p.id, p.logger)
}

func (p *rwFileProvider) Close() {
	p.session.Close(p.logger)
}

func (p *rwFileProvider) IsComplete() bool {
	file, err := p.session.AcquireFile()
	if err != nil {
		return false
	}
	defer p.session.ReleaseFile()

	complete := file.IsComplete()
	if complete {
		p.session.MarkComplete()
	}
	return complete
}

func (p *rwFileProvider) IsForceResolving() bool {
	return p.session.IsForceExpiration()
}

func (p *rwFileProvider) DownloadedSize() int64 {
	file, err := p.session.AcquireFile()
	if err != nil {
		return 0
	}
	defer p.session.ReleaseFile()

	return file.GetDownloadedBytes()
}

func (p *rwFileProvider) CurrentCursor() (int64, error) {
	file, err := p.session.AcquireFile()
	if err != nil {
		return 0, err
	}
	defer p.session.ReleaseFile()

	file.MarkDownloading()
	offset := file.GetDownloadOffset()

	p.session.Logger().InfoLnf("Download %s start from %d, (total %d)", p.id, offset, file.TotalLen())

	return offset, nil
}

func (p *rwFileProvider) SeekStart() error {
	file, err := p.session.AcquireFile()
	if err != nil {
		return err
	}
	defer p.session.ReleaseFile()

	return file.SeekStart()
}

var _ LocalProvider = &rwFileProvider{}

func NewRWFileProvider(id string, session types.CDNFileSession, logger utils.LoggerImpl) LocalProvider {
	return &rwFileProvider{id, session, logger}
}
