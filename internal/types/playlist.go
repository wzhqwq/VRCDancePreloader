package types

import (
	"io"

	"fyne.io/fyne/v2"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
)

type PlayItemI interface {
	UpdateProgress(downloaded int64)
	UpdatePreloadStatus(status constants.Status)
	UpdatePlayStatus(status constants.Status)
	UpdateIndex(index int)
	UpdateSize(size int64)

	Download()
	ToReader() (io.ReadSeekCloser, error)
	Play(now float64)

	Render() *PlayItemRendered
	GetInfo() *PlayItemInfo
	IsDisposed() bool

	Detach()
	Dispose()
}

type PlayItemInfo struct {
	ID    int
	Title string
	Group string
	Adder string
	Size  int64
	URL   string
}

type PlayItemRendered struct {
	// static
	ID    int
	Title string
	Group string
	Adder string

	OriginalVideoURL string

	// dynamic
	Status           string
	StatusColor      fyne.ThemeColorName
	Size             string
	DownloadProgress float64
	Index            int
	ErrorText        string
	IsDownloading    bool
	IsPlaying        bool
	PlayProgress     float64
	PlayTimeText     string
}
