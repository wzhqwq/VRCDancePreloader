package cache

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

type DownloadState struct {
	Downloaded int64
	Item       types.PlayItemI
	Tracker    *progress.Tracker
	Mutex      sync.Mutex
}

var pw progress.Writer = progress.NewWriter()

var downloadStateMap = make(map[int]*DownloadState)
var stateMapMutex = sync.Mutex{}

var downloadCh chan int
var prioritizedID int

func (ds *DownloadState) Write(p []byte) (int, error) {
	if ds.Item.IsDisposed() {
		return 0, io.EOF
	}

	n := len(p)
	ds.Downloaded += int64(n)
	ds.Item.UpdateProgress(ds.Downloaded)
	ds.Tracker.Increment(int64(n))
	return n, nil
}

func progressiveDownload(body io.ReadCloser, file *os.File, ds *DownloadState) error {
	ds.Item.UpdateStatus(constants.Downloading)

	downloadCh <- ds.Item.GetInfo().ID
	defer func() {
		<-downloadCh
	}()
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(file, io.TeeReader(body, ds))
	if err != nil {
		ds.Tracker.MarkAsErrored()
		return err
	}
	ds.Tracker.MarkAsDone()

	return nil
}

func Download(item types.PlayItemI) error {
	i := item.GetInfo()
	// get mutex for the id
	stateMapMutex.Lock()
	ds, ok := downloadStateMap[i.ID]
	if !ok {
		pt := &progress.Tracker{
			Message: fmt.Sprintf("Downloading %s", i.Title),
			Total:   int64(i.Size),
			Units:   progress.UnitsBytes,
		}
		pw.AppendTracker(pt)
		defer func() {
			pt.MarkAsDone()
		}()

		ds = &DownloadState{
			Item:    item,
			Tracker: pt,
			Mutex:   sync.Mutex{},
		}
		downloadStateMap[i.ID] = ds
	}
	stateMapMutex.Unlock()

	// lock the mutex to force single download
	ds.Mutex.Lock()
	defer ds.Mutex.Unlock()

	// check if file is already downloaded
	if size := getCacheSize(i.ID); size > 0 {
		item.UpdateSize(size)
		return nil
	}

	// open temp file
	tempFile := openTempFile(i.ID)
	if tempFile == nil {
		return fmt.Errorf("failed to open temp_%d.mp4", i.ID)
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	// download the file
	item.UpdateStatus(constants.Requesting)
	resp, err := http.Get(i.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// get size of the file to be downloaded
	item.UpdateSize(resp.ContentLength)

	// download
	err = progressiveDownload(resp.Body, tempFile, ds)
	if err != nil {
		return err
	}

	// copy to file
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return err
	}
	file := OpenCache(i.ID)
	if file == nil {
		return fmt.Errorf("failed to open %d.mp4", i.ID)
	}
	io.Copy(file, tempFile)

	// close and reopen
	closeCache(i.ID)
	OpenCache(i.ID)

	item.UpdateProgress(i.Size)

	delete(downloadStateMap, i.ID)

	return nil
}

func Prioritize(id int) {
	prioritizedID = id
}
