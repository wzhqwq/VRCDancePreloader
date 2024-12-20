package cache

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/requesting"
)

type DownloadState struct {
	sync.Mutex

	ID string

	TotalSize      int64
	DownloadedSize int64

	Done    bool
	Pending bool
	Error   error

	FinalURL string

	StateCh    chan *DownloadState
	CancelCh   chan bool
	PriorityCh chan int
}

func (ds *DownloadState) Write(p []byte) (int, error) {
	select {
	case <-ds.CancelCh:
		return 0, io.EOF
	default:
		ds.BlockIfPending()
		n := len(p)
		ds.DownloadedSize += int64(n)
		ds.StateCh <- ds
		return n, nil
	}
}
func (ds *DownloadState) BlockIfPending() bool {
	select {
	case p := <-ds.PriorityCh:
		if dm.CanDownload(p) {
			if ds.Pending {
				ds.Pending = false
				ds.StateCh <- ds
			}
			log.Printf("%s now can continue download\n", ds.ID)
			return true
		} else {
			log.Printf("%s is now pending, priority %d\n", ds.ID, p)
			if !ds.Pending {
				ds.Pending = true
				ds.StateCh <- ds
			}
		}
	default:
		return true
	}
	for {
		select {
		case <-ds.CancelCh:
			return false
		case p := <-ds.PriorityCh:
			if dm.CanDownload(p) {
				if ds.Pending {
					ds.Pending = false
					ds.StateCh <- ds
				}
				log.Printf("%s now can continue download\n", ds.ID)
				return true
			} else {
				log.Printf("%s is now pending, priority %d\n", ds.ID, p)
				if !ds.Pending {
					ds.Pending = true
					ds.StateCh <- ds
				}
			}
		}
	}
}
func (ds *DownloadState) unlockAndNotify() {
	ds.Unlock()
	ds.StateCh <- ds
}

func progressiveDownload(body io.ReadCloser, file *os.File, ds *DownloadState) error {
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(file, io.TeeReader(body, ds))
	if err != nil {
		return err
	}

	return nil
}

func Download(id, url string) *DownloadState {
	ds := dm.CreateOrGetState(id)
	go func() {
		// Lock the state so that there is always only one download happening
		ds.Lock()
		defer ds.unlockAndNotify()

		// Clear error every time we start downloading
		ds.Error = nil

		if ds.Done {
			return
		}

		// Otherwise we start downloading, the total size is unknown
		ds.TotalSize = 0
		ds.BlockIfPending()

		// Open temp file
		tempFile := openTempFile(id)
		if tempFile == nil {
			ds.Error = fmt.Errorf("failed to open %s.mp4.dl", id)
			return
		}
		defer func() {
			tempFile.Close()
			os.Remove(tempFile.Name())
		}()

		// Download the file
		if ds.FinalURL != "" {
			url = ds.FinalURL
		}
		resp, err := requesting.RequestVideo(url)
		if err != nil {
			ds.Error = err
			return
		}
		defer resp.Body.Close()

		// Save the size of the file to be downloaded
		ds.TotalSize = resp.ContentLength
		if resp.Header.Get("Location") != "" {
			// Save the final URL if it's redirected
			ds.FinalURL = resp.Header.Get("Location")
		}

		// TODO: Replace them after implementing resuming download halfway
		_, err = tempFile.Seek(0, 0)
		if err != nil {
			ds.Error = err
			return
		}
		ds.DownloadedSize = 0

		// Notify about the total size and that the request header is done
		ds.StateCh <- ds

		// Copy the body to the file, which will also update the download progress
		err = progressiveDownload(resp.Body, tempFile, ds)
		if err != nil {
			ds.Error = err
			return
		}

		// Copy the temp file to the cache file
		// If the progress can be stopped gracefully, the temp file could be deleted
		file := OpenCache(id)
		if file == nil {
			ds.Error = fmt.Errorf("failed to open %s.mp4", id)
			return
		}
		_, err = tempFile.Seek(0, 0)
		if err != nil {
			ds.Error = err
			return
		}
		_, err = io.Copy(file, tempFile)
		if err != nil {
			ds.Error = err
			return
		}

		// Close the cache file so that all data is written to disk
		closeCache(id)

		// Mark the download as done and update the priorities
		ds.Done = true
		dm.UpdatePriorities()
	}()

	return ds
}

func CancelDownload(id string) {
	dm.CancelDownload(id)
}

func Prioritize(id string) {
	dm.Prioritize(id)
}
