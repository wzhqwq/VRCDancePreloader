package playlist

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
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
	PreloadStatus   constants.Status
	PlayStatus      constants.Status
	Progress        float64
	DownloadedBytes int64
	Index           int
	Error           error
	Now             float64

	// private
	downloadingMutex sync.Mutex

	dirty       bool
	disposed    bool
	updateMutex sync.Mutex
}

func NewPlayItem(title, group, adder, url string, id, duration int) *PlayItem {
	preloadStatus := constants.Pending
	if id >= 0 {
		url = fmt.Sprintf("http://jd.pypy.moe/api/v1/videos/%d.mp4", id)
	} else {
		preloadStatus = constants.Ignored
	}

	return &PlayItem{
		ID:       id,
		Title:    title,
		Group:    group,
		Adder:    adder,
		URL:      url,
		Duration: duration,
		Index:    -1,

		PreloadStatus: preloadStatus,
		PlayStatus:    constants.Pending,
	}
}

func (i *PlayItem) IsDisposed() bool {
	return i.disposed
}

func (i *PlayItem) Download() {
	if i.PreloadStatus != constants.Pending {
		return
	}

	if i.URL == "" {
		i.Error = fmt.Errorf("no url")
		i.UpdatePreloadStatus(constants.Failed)
		return
	}

	i.downloadingMutex.Lock()
	defer i.downloadingMutex.Unlock()

	// download the file, and keep track of progress
	err := cache.Download(i)
	if err != nil {
		i.Error = err
		i.UpdatePreloadStatus(constants.Failed)
		log.Println("Failed to download", err)
		return
	}

	i.UpdatePreloadStatus(constants.Downloaded)
	log.Println("Downloaded", i.Title)
}

func (i *PlayItem) Play(now float64) {
	if i.PlayStatus == constants.Ended {
		return
	}
	i.UpdatePlayStatus(constants.Playing)
	i.Now = now
}

type nopCloserRS struct {
	io.Reader
	io.Seeker
}

func (nopCloserRS) Close() error { return nil }

func (i *PlayItem) ToReader() (io.ReadSeekCloser, error) {
	if i.PreloadStatus == constants.Pending || i.PreloadStatus == constants.Downloading {
		i.Download()
		if i.PreloadStatus == constants.Failed {
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
func (i *PlayItem) Dispose() {
	i.disposed = true
	i.Detach()
}
