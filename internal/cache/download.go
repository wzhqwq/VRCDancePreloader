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

	TotalSize  int64
	Downloaded int64
	Done       bool
	Pending    bool
	Error      error

	StateCh    chan *DownloadState
	CancelCh   chan bool
	PriorityCh chan int

	tracker *progress.Tracker
}

var pw progress.Writer = progress.NewWriter()

func (ds *DownloadState) Write(p []byte) (int, error) {
	select {
	case <-ds.CancelCh:
		return 0, io.EOF
	default:
		n := len(p)
		ds.Downloaded += int64(n)
		ds.tracker.Increment(int64(n))
		ds.StateCh <- ds
		return n, nil
	}
}
func (ds *DownloadState) BlockIfPending() {
	for {
		select {
		case p := <-ds.PriorityCh:
			if dm.CanDownload(p) {
				if ds.Pending {
					ds.Pending = false
					ds.StateCh <- ds
				}
				return
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
		ds.tracker.MarkAsErrored()
		return err
	}
	ds.tracker.MarkAsDone()

	return nil
}

func Download(id int, url string) *DownloadState {
	ds := dm.CreateOrGetState(id)
	go func() {
		ds.Lock()
		defer ds.unlockAndUpdate()

		if ds.Done {
			return
		}

		// open temp file
		tempFile := openTempFile(id)
		if tempFile == nil {
			ds.Error = fmt.Errorf("failed to open temp_%d.mp4", id)
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
		ds.StateCh <- ds

		if ds.tracker == nil {
			pt := &progress.Tracker{
				Message: fmt.Sprintf("Downloading %s", url),
				Total:   int64(ds.TotalSize),
				Units:   progress.UnitsBytes,
			}
			pw.AppendTracker(pt)
			ds.tracker = pt
		}

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
			ds.Error = fmt.Errorf("failed to open %d.mp4", id)
			return
		}
		io.Copy(file, tempFile)
		closeCache(id)

		ds.Done = true
		dm.UpdatePriorities()
	}()

	return ds
}
