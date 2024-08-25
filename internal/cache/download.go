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

var pw progress.Writer = progress.NewWriter()

type WriteCounter struct {
	Total   int64
	item    types.PlayItemI
	tracker *progress.Tracker
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += int64(n)
	wc.item.UpdateProgress(wc.Total)
	wc.tracker.Increment(int64(n))
	return n, nil
}

func progressiveDownload(body io.ReadCloser, file *os.File, item types.PlayItemI) error {
	item.UpdateStatus(constants.Downloading)
	i := item.GetInfo()
	pt := &progress.Tracker{
		Message: fmt.Sprintf("Downloading %s", i.Title),
		Total:   int64(i.Size),
		Units:   progress.UnitsBytes,
	}
	pw.AppendTracker(pt)
	counter := &WriteCounter{item: item, tracker: pt}

	// Write the body to file, while showing progress of the download
	_, err := io.Copy(file, io.TeeReader(body, counter))
	if err != nil {
		pt.MarkAsErrored()
		return err
	}
	pt.MarkAsDone()

	return nil
}

func Download(item types.PlayItemI) error {
	i := item.GetInfo()
	// get mutex for the id
	mapMutex.Lock()
	mutex, ok := mutexMap[i.ID]
	if !ok {
		mutex = &sync.Mutex{}
		mutexMap[i.ID] = mutex
	}
	mapMutex.Unlock()

	// lock the mutex to force single download
	mutex.Lock()
	defer mutex.Unlock()

	// open file from cache
	file := OpenCache(i.ID)
	if file == nil {
		return fmt.Errorf("failed to open %d.mp4", i.ID)
	}

	// get file size
	stat, statErr := file.Stat()
	if statErr != nil {
		return statErr
	}

	// check if file is already downloaded
	if size := stat.Size(); size > 0 {
		item.UpdateSize(size)
		return nil
	}

	// download the file
	item.UpdateStatus(constants.Requesting)
	resp, err := http.Get(i.URL)
	if err != nil {
		RemoveCache(i.ID)
		return err
	}
	defer resp.Body.Close()

	// get size of the file to be downloaded
	item.UpdateSize(resp.ContentLength)

	err = progressiveDownload(resp.Body, file, item)
	if err != nil {
		RemoveCache(i.ID)
		return err
	}

	// close and reopen
	closeCache(i.ID)
	OpenCache(i.ID)

	item.UpdateProgress(i.Size)

	return nil
}
