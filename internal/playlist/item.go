package playlist

import (
	"fmt"
	"io"
	"log"
	"math"
	"sync"

	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

type PlayItem struct {
	// static
	ID    int
	Title string
	Group string
	Adder string

	URL      string
	Size     int64
	Duration int

	// dynamic
	Status          constants.Status
	Progress        float64
	DownloadedBytes int64
	Index           int
	Error           error

	// private
	downloadingMutex sync.Mutex

	dirty       bool
	updateMutex sync.Mutex
}

func NewPlayItem(title, group, adder, url string, id, duration, index int) *PlayItem {
	status := constants.Pending
	if id >= 0 {
		url = fmt.Sprintf("http://jd.pypy.moe/api/v1/videos/%d.mp4", id)
	} else {
		status = constants.Ignored
	}

	return &PlayItem{
		ID:       id,
		Title:    title,
		Group:    group,
		Adder:    adder,
		URL:      url,
		Duration: duration,
		Status:   status,

		Index: index,
	}
}

func (i *PlayItem) UpdateProgress(downloaded int64) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	progress := float64(downloaded) / float64(i.Size)
	if math.Abs(progress-i.Progress) < 0.001 {
		return
	}
	i.Progress = progress
	i.DownloadedBytes = downloaded

	i.updateGui()
}
func (i *PlayItem) UpdateStatus(status constants.Status) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	if i.Status == status {
		return
	}
	i.Status = status

	switch status {
	case constants.Ended:
		i.Detach()
	case constants.Failed:
		i.Detach()
	case constants.Playing:
		CriticalUpdate()
	case constants.Downloaded:
		CriticalUpdate()
	}

	i.updateGui()
}
func (i *PlayItem) UpdateIndex(index int) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	if i.Index == index {
		return
	}
	i.Index = index

	i.updateGui()
}
func (i *PlayItem) UpdateSize(size int64) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	if i.Size == size {
		return
	}
	i.Size = size

	i.updateGui()
}
func (i *PlayItem) updateGui() {
	if !i.dirty {
		gui.UpdatePlayItem(i)
		i.dirty = true
	}
}

func (i *PlayItem) Render() *types.PlayItemRendered {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	errorText := ""
	if i.Status == constants.Failed && i.Error != nil {
		errorText = i.Error.Error()
	}

	sizeText := i18n.T("placeholder_unknown_size")
	if i.Size > 0 {
		sizeText = utils.PrettyByteSize(i.Size)
	}
	if i.Status == constants.Downloading {
		sizeText = utils.PrettyByteSize(i.DownloadedBytes) + "/" + sizeText
	}

	color := theme.ColorNamePlaceHolder
	switch i.Status {
	case constants.Requesting, constants.Downloading, constants.Playing:
		color = theme.ColorNamePrimary
	case constants.Downloaded:
		color = theme.ColorNameSuccess
	case constants.Failed:
		color = theme.ColorNameError
	}

	i.dirty = false
	return &types.PlayItemRendered{
		ID:    i.ID,
		Title: i.Title,
		Group: i.Group,
		Adder: i.Adder,

		Status:        i18n.T("status_" + string(i.Status)),
		StatusColor:   color,
		Size:          sizeText,
		Progress:      i.Progress,
		ErrorText:     errorText,
		Index:         i.Index,
		IsDownloading: i.Status == constants.Downloading,
	}
}
func (i *PlayItem) GetInfo() *types.PlayItemInfo {
	return &types.PlayItemInfo{
		ID:    i.ID,
		Title: i.Title,
		Group: i.Group,
		Adder: i.Adder,
		Size:  i.Size,
		URL:   i.URL,
	}
}

func (i *PlayItem) Download() {
	if i.Status != constants.Pending {
		return
	}

	if i.URL == "" {
		i.Error = fmt.Errorf("no url")
		i.UpdateStatus(constants.Failed)
		return
	}

	i.downloadingMutex.Lock()
	defer i.downloadingMutex.Unlock()

	// download the file, and keep track of progress
	err := cache.Download(i)
	if err != nil {
		i.Error = err
		i.UpdateStatus(constants.Failed)
		log.Println("Failed to download", err)
		return
	}

	i.UpdateStatus(constants.Downloaded)
	log.Println("Downloaded", i.Title)
}

type nopCloserRS struct {
	io.Reader
	io.Seeker
}

func (nopCloserRS) Close() error { return nil }

func (i *PlayItem) ToReader() (io.ReadSeekCloser, error) {
	if i.Status == constants.Pending || i.Status == constants.Downloading {
		i.Download()
		if i.Status == constants.Failed {
			return nil, fmt.Errorf("failed to download %s", i.Title)
		}
	}

	file := cache.OpenCache(i.ID)
	if file == nil {
		return nil, fmt.Errorf("failed to load %s", i.Title)
	}

	_, err := file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	return nopCloserRS{file, file}, nil
}

func (i *PlayItem) Detach() {
	cache.DetachCache(i.ID)
}
