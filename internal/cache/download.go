package cache

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"
)

type DownloadState struct {
	sync.Mutex

	TotalSize      int64
	DownloadedSize int64
	Done           bool
	Pending        bool
	Error          error

	StateCh    chan *DownloadState
	CancelCh   chan bool
	PriorityCh chan int
}

var pw progress.Writer = progress.NewWriter()

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
				return true
			} else {
				if !ds.Pending {
					ds.Pending = true
					ds.StateCh <- ds
				}
			}
		}
	}
}
func (ds *DownloadState) unlockAndUpdate() {
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
		ds.Lock()
		defer ds.unlockAndUpdate()

		ds.Error = nil

		if ds.Done {
			return
		}

		// open temp file
		tempFile := openTempFile(id)
		if tempFile == nil {
			ds.Error = fmt.Errorf("failed to open %s.mp4.dl", id)
			return
		}
		defer func() {
			tempFile.Close()
			os.Remove(tempFile.Name())
		}()

		// download the file
		resp, err := http.Get(url)
		if err != nil {
			ds.Error = err
			return
		}
		defer resp.Body.Close()
		// get size of the file to be downloaded
		ds.TotalSize = resp.ContentLength
		// TODO: Remove after implementing resuming download halfway
		ds.DownloadedSize = 0
		ds.StateCh <- ds

		// download
		err = progressiveDownload(resp.Body, tempFile, ds)
		if err != nil {
			ds.Error = err
			return
		}

		// copy to file
		_, err = tempFile.Seek(0, 0)
		if err != nil {
			ds.Error = err
			return
		}
		file := OpenCache(id)
		if file == nil {
			ds.Error = fmt.Errorf("failed to open %s.mp4", id)
			return
		}
		io.Copy(file, tempFile)
		closeCache(id)

		ds.Done = true
		dm.UpdatePriorities()
	}()

	return ds
}
