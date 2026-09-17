package task

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

// fakeLocal is a minimal LocalProvider that keeps everything in memory. It is
// used by the task level tests; the tests that need real files use the
// rw_file backed provider instead.
type fakeLocal struct {
	mu sync.Mutex

	written  int64
	complete bool

	opened int
	closed int
}

func (f *fakeLocal) Open() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.opened++
	return nil
}

func (f *fakeLocal) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed++
}

func (f *fakeLocal) IsComplete() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.complete
}

func (f *fakeLocal) setComplete(complete bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.complete = complete
}

func (f *fakeLocal) IsForceResolving() bool { return false }

func (f *fakeLocal) DownloadedSize() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.written
}

func (f *fakeLocal) CurrentCursor() (int64, error) {
	return f.DownloadedSize(), nil
}

func (f *fakeLocal) SeekStart() error { return nil }

func (f *fakeLocal) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.written += int64(len(p))
	return len(p), nil
}

// fakeRemote is a scriptable RemoteProvider.
//
// GetDownloadStream behaviour, in order of precedence:
//   - when blockStream is set, it parks until the context is done (or until the
//     test closes blockStream) and then returns the cancellation cause, which is
//     what a real provider does when the attempt scope is aborted;
//   - otherwise it returns streamErr when set;
//   - otherwise it hands out an empty body, which the download loop treats as a
//     restartable EOF.
type fakeRemote struct {
	mu sync.Mutex

	resolveCalls int
	streamCalls  int

	resolveErr error
	totalSize  int64

	streamErr   error
	blockStream chan struct{}

	// entered receives a token for every GetDownloadStream call that got past
	// the entry bookkeeping (non blocking, buffered).
	entered chan struct{}

	// streamCtxErr records the state of the context handed to each
	// GetDownloadStream call, taken at entry.
	streamCtxErr []error
}

func newFakeRemote() *fakeRemote {
	return &fakeRemote{
		entered: make(chan struct{}, 64),
	}
}

func (f *fakeRemote) WaitResolving(_ context.Context, _ func(interactive.RemoteStatus)) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.resolveCalls++

	if f.resolveErr != nil {
		return 0, f.resolveErr
	}

	size := f.totalSize
	if size == 0 {
		size = 1024
	}
	return size, nil
}

func (f *fakeRemote) GetDownloadStream(_ int64, ctx context.Context) (StreamInfo, error) {
	f.mu.Lock()
	f.streamCalls++
	err := f.streamErr
	block := f.blockStream
	f.streamCtxErr = append(f.streamCtxErr, ctx.Err())
	f.mu.Unlock()

	select {
	case f.entered <- struct{}{}:
	default:
	}

	if block != nil {
		select {
		case <-ctx.Done():
			return StreamInfo{}, context.Cause(ctx)
		case <-block:
			return StreamInfo{}, context.Cause(ctx)
		}
	}

	if err != nil {
		return StreamInfo{}, err
	}

	return StreamInfo{
		Rc:             io.NopCloser(strings.NewReader("")),
		Length:         0,
		RangeAvailable: true,
	}, nil
}

func (f *fakeRemote) counts() (resolve, stream int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.resolveCalls, f.streamCalls
}

func (f *fakeRemote) ctxErrs() []error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]error(nil), f.streamCtxErr...)
}
