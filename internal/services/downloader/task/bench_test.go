package task

import (
	"sync"
	"testing"
)

// The download loop calls Write once per chunk of the body (io.Copy uses a 32KB
// buffer), so this is the hot path of the whole engine — and it is the path the
// state mutex guards. These benchmarks exist to keep that guard honest: it has
// to stay noise against the cost of moving 32KB over the network.
//
// Run with:
//
//	go test -bench=. -benchmem -run=^$ ./internal/services/downloader/task/

const benchChunk = 32 * 1024

func newBenchTask() *Task {
	return NewTask("bench", newFakeRemote(), &fakeLocal{})
}

// BenchmarkTaskWrite is the per chunk cost of the real download loop, including
// everything Task.Write does (gate entry, progress accounting, event).
func BenchmarkTaskWrite(b *testing.B) {
	t := newBenchTask()
	chunk := make([]byte, benchChunk)

	b.SetBytes(benchChunk)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := t.Write(chunk); err != nil {
			b.Fatalf("write: %v", err)
		}
	}
}

// BenchmarkTaskAddBytes isolates the part the state mutex actually guards: the
// downloaded counter, the eta window and the event notification.
func BenchmarkTaskAddBytes(b *testing.B) {
	t := newBenchTask()

	b.SetBytes(benchChunk)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.addBytes(benchChunk)
	}
}

// BenchmarkTaskReadUncontended is what a reader pays when the download goroutine
// happens not to be inside the critical section.
func BenchmarkTaskReadUncontended(b *testing.B) {
	t := newBenchTask()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = t.DownloadedSize()
		_, _ = t.StateAndError()
	}
}

// BenchmarkTaskReadContended is the pattern the manager, song and the GUI
// actually run in: reading while the download goroutine keeps writing.
//
// The writer here is an unpaced loop, so this is a worst case — in production
// the writer is paced by the network (one call per 32KB received), which is
// orders of magnitude slower than this loop.
func BenchmarkTaskReadContended(b *testing.B) {
	t := newBenchTask()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-stop:
				return
			default:
				t.addBytes(benchChunk)
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = t.DownloadedSize()
	}
	b.StopTimer()

	close(stop)
	wg.Wait()
}
