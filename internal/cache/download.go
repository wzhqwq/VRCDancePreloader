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
	err = progressiveDownload(resp.Body, tempFile, item)
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

	return nil
}
