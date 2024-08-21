package playlist

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

type PlayItem struct {
	ID    int
	Title string
	Group string
	Adder string

	URL      string
	Size     int
	Duration int

	Status   constants.Status
	Progress float64

	DownloadWaitingChs      []chan struct{}
	DownloadWaitingChsMutex sync.Mutex

	PlayWaitingChs      []chan struct{}
	PlayWaitingChsMutex sync.Mutex

	Index int
	Dirty bool
}

func NewPlayItem(title, group, adder, url string, id, duration int) *PlayItem {
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

		DownloadWaitingChs: make([]chan struct{}, 0),
		PlayWaitingChs:     make([]chan struct{}, 0),
	}
}

func (i *PlayItem) UpdateProgress(progress float64) {
	if i.Progress == progress {
		return
	}
	i.Progress = progress
	i.Dirty = true
}
func (i *PlayItem) UpdateStatus(status constants.Status) {
	changed := i.Status != status
	i.Status = status

	if changed {
		i.Dirty = true

		switch status {
		case constants.Ended:
			i.Detach()
		case constants.Failed:
			i.Detach()
		case constants.Playing:
			i.PlayWaitingChsMutex.Lock()
			for _, ch := range i.PlayWaitingChs {
				close(ch)
			}
			i.PlayWaitingChs = make([]chan struct{}, 0)
			i.PlayWaitingChsMutex.Unlock()
		}
	}
}
func (i *PlayItem) UpdateIndex(index int) {
	if i.Index == index {
		return
	}
	i.Index = index
	i.Dirty = true
	gui.UpdatePlayItemCh <- i
}

func (i *PlayItem) GetRendered() *types.PlayItemRendered {
	return &types.PlayItemRendered{
		ID:       i.ID,
		Title:    i.Title,
		Group:    i.Group,
		Adder:    i.Adder,
		Status:   i18n.T("status_" + string(i.Status)),
		Progress: i.Progress,
		Index:    i.Index,
	}
}

func (i *PlayItem) Download() {
	if i.Status != constants.Pending {
		return
	}

	if i.URL == "" {
		i.UpdateStatus(constants.Failed)
		return
	}

	file := cache.RequestCache(i.ID)
	if file == nil {
		i.UpdateStatus(constants.Failed)
		return
	}
	stat, statErr := file.Stat()
	if statErr == nil && stat.Size() > 0 {
		i.UpdateStatus(constants.Downloaded)
		i.Size = int(stat.Size())
		return
	} else {
		i.UpdateStatus(constants.Downloading)
	}

	resp, err := http.Get(i.URL)
	if err != nil {
		i.UpdateStatus(constants.Failed)
		return
	}
	defer resp.Body.Close()

	// get size of the file to be downloaded
	i.Size = int(resp.ContentLength)
	log.Println("Downloading", i.Title, "size:", i.Size)

	// file already downloaded
	if statErr == nil && stat.Size() == int64(i.Size) {
		i.UpdateStatus(constants.Downloaded)
		return
	}

	// download the file, and keep track of progress
	err = ProgressiveDownload(resp.Body, file, i)
	if err != nil {
		i.UpdateStatus(constants.Failed)
		log.Println("constants.Failed to download", err)
		return
	}

	cache.FlushCache(i.ID)
	i.UpdateStatus(constants.Downloaded)

	log.Println("constants.Downloaded", i.Title)

	// notify waiting readers
	i.DownloadWaitingChsMutex.Lock()
	for _, ch := range i.DownloadWaitingChs {
		close(ch)
	}
	i.DownloadWaitingChs = make([]chan struct{}, 0)
	i.DownloadWaitingChsMutex.Unlock()
}

type nopCloserRS struct {
	io.Reader
	io.Seeker
}

func (nopCloserRS) Close() error { return nil }

func (i *PlayItem) ToReader() (io.ReadSeekCloser, error) {
	if i.Status == constants.Downloading {
		ch := make(chan struct{})
		i.DownloadWaitingChsMutex.Lock()
		i.DownloadWaitingChs = append(i.DownloadWaitingChs, ch)
		i.DownloadWaitingChsMutex.Unlock()
		<-ch
	}
	if i.Status == constants.Pending {
		i.Download()
		if i.Status == constants.Failed {
			return nil, fmt.Errorf("failed to download %s", i.Title)
		}
	}

	file := cache.RequestCache(i.ID)
	if file == nil {
		return nil, fmt.Errorf("failed to load %s", i.Title)
	}

	_, err := file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	return nopCloserRS{file, file}, nil
}

func (i *PlayItem) WaitForPlay() {
	if i.Status == constants.Playing {
		return
	}
	ch := make(chan struct{})
	i.PlayWaitingChsMutex.Lock()
	i.PlayWaitingChs = append(i.PlayWaitingChs, ch)
	i.PlayWaitingChsMutex.Unlock()
	<-ch
}

func (i *PlayItem) Detach() {
	cache.DetachCache(i.ID)
}
