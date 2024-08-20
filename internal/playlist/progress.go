package playlist

import (
	"fmt"
	"io"
	"os"

	"github.com/jedib0t/go-pretty/v6/progress"
)

var pw progress.Writer = progress.NewWriter()

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total   uint64
	item    *PlayItem
	tracker *progress.Tracker
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.item.UpdateProgress(float64(wc.Total) / float64(wc.item.Size) * 100)
	wc.tracker.Increment(int64(n))
	return n, nil
}

func ProgressiveDownload(body io.ReadCloser, file *os.File, item *PlayItem) error {
	item.UpdateStatus(Downloading)
	pt := &progress.Tracker{
		Message: fmt.Sprintf("Downloading %s", item.Title),
		Total:   int64(item.Size),
		Units:   progress.UnitsBytes,
	}
	pw.AppendTracker(pt)
	counter := &WriteCounter{item: item, tracker: pt}

	defer file.Close()

	// Write the body to file, while showing progress of the download
	_, err := io.Copy(file, io.TeeReader(body, counter))
	if err != nil {
		return err
	}
	pt.MarkAsDone()

	return nil
}
