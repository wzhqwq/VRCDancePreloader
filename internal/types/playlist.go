package types

import (
	"io"

	"fyne.io/fyne/v2"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
)

type PlayItemI interface {
	UpdateProgress(downloaded int64)
	UpdateStatus(status constants.Status)
	UpdateIndex(index int)
	UpdateSize(size int64)
	Download()
	ToReader() (io.ReadSeekCloser, error)
	Render() *PlayItemRendered
	GetInfo() *PlayItemInfo
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

	// dynamic
	Status        string
	StatusColor   fyne.ThemeColorName
	Size          string
	Progress      float64
	Index         int
	ErrorText     string
	IsDownloading bool
}
