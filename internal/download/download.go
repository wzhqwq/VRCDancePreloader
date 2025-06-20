package download

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"io"
	"log"
	"sync"
)

type DownloadState struct {
	sync.Mutex

	ID string

	cacheEntry cache.Entry

	TotalSize      int64
	DownloadedSize int64

	Done    bool
	Pending bool
	Error   error

	StateCh    chan *DownloadState
	CancelCh   chan bool
	PriorityCh chan int
}

func (ds *DownloadState) Write(p []byte) (int, error) {
	select {
	case <-ds.CancelCh:
		return 0, io.ErrClosedPipe
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

func (ds *DownloadState) progressiveDownload(body io.ReadCloser, writer io.Writer) error {
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(writer, io.TeeReader(body, ds))
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func Download(id string) *DownloadState {
	ds := dm.CreateOrGetState(id)
	if ds == nil {
		return nil
	}
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

		entry := ds.cacheEntry
		ds.TotalSize = entry.TotalLen()

		body, err := entry.GetDownloadBody()
		if err != nil {
			ds.Error = err
			log.Println("Start Downloading error")
			return
		}
		defer body.Close()

		ds.DownloadedSize = 0

		// Notify about the total size and that the request header is done
		ds.StateCh <- ds

		// Copy the body to the file, which will also update the download progress
		err = ds.progressiveDownload(body, entry)
		if err != nil {
			ds.Error = err
			log.Println("Downloading error")
			return
		}

		err = entry.Save()
		if err != nil {
			ds.Error = err
			log.Println("Saving error")
			return
		}

		// Mark the download as done and update the priorities
		ds.Done = true
		dm.UpdatePriorities()
	}()

	return ds
}
