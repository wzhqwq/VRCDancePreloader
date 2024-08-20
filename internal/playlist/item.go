package playlist

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
)

type Status string

const (
	Downloading Status = "downloading"
	Downloaded  Status = "downloaded"
	Failed      Status = "failed"
	Pending     Status = "pending"
	Playing     Status = "playing"
	Ended       Status = "ended"
)

type PlayItem struct {
	ID    int
	Title string
	Group string
	Adder string

	URL      string
	Size     int
	Duration int

	Status   Status
	Progress float64

	DownloadWaitingChs      []chan struct{}
	DownloadWaitingChsMutex sync.Mutex

	PlayWaitingChs      []chan struct{}
	PlayWaitingChsMutex sync.Mutex
}

func NewPlayItem(title, group, adder, url string, id, duration int) *PlayItem {
	if id >= 0 {
		url = fmt.Sprintf("http://jd.pypy.moe/api/v1/videos/%d.mp4", id)
	}
	return &PlayItem{
		ID:       id,
		Title:    title,
		Group:    group,
		Adder:    adder,
		URL:      url,
		Duration: duration,
		Status:   Pending,

		DownloadWaitingChs: make([]chan struct{}, 0),
		PlayWaitingChs:     make([]chan struct{}, 0),
	}
}

func (i *PlayItem) UpdateProgress(progress float64) {
	i.Progress = progress
}
func (i *PlayItem) UpdateStatus(status Status) {
	changed := i.Status != status
	i.Status = status

	if changed {
		switch status {
		case Ended:
			i.Detach()
		case Failed:
			i.Detach()
		case Playing:
			i.PlayWaitingChsMutex.Lock()
			for _, ch := range i.PlayWaitingChs {
				close(ch)
			}
			i.PlayWaitingChs = make([]chan struct{}, 0)
			i.PlayWaitingChsMutex.Unlock()
		}
	}
}

func (i *PlayItem) Download() {
	if i.Status != Pending {
		return
	}

	if i.URL == "" {
		i.UpdateStatus(Failed)
		return
	}

	file := cache.RequestCache(i.ID)
	if file == nil {
		i.UpdateStatus(Failed)
		return
	}
	stat, statErr := file.Stat()
	if statErr == nil && stat.Size() > 0 {
		i.UpdateStatus(Downloaded)
		i.Size = int(stat.Size())
		return
	} else {
		i.UpdateStatus(Downloading)
	}

	resp, err := http.Get(i.URL)
	if err != nil {
		i.UpdateStatus(Failed)
		return
	}
	defer resp.Body.Close()

	// get size of the file to be downloaded
	i.Size = int(resp.ContentLength)
	log.Println("Downloading", i.Title, "size:", i.Size)

	// file already downloaded
	if statErr == nil && stat.Size() == int64(i.Size) {
		i.UpdateStatus(Downloaded)
		return
	}

	// download the file, and keep track of progress
	err = ProgressiveDownload(resp.Body, file, i)
	if err != nil {
		i.UpdateStatus(Failed)
		log.Println("Failed to download", err)
		return
	}

	cache.FlushCache(i.ID)
	i.UpdateStatus(Downloaded)

	log.Println("Downloaded", i.Title)

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
	if i.Status == Downloading {
		ch := make(chan struct{})
		i.DownloadWaitingChsMutex.Lock()
		i.DownloadWaitingChs = append(i.DownloadWaitingChs, ch)
		i.DownloadWaitingChsMutex.Unlock()
		<-ch
	}
	if i.Status == Pending {
		i.Download()
		if i.Status == Failed {
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
	if i.Status == Playing {
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
