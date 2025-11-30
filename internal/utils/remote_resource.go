package utils

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

var ErrCanceled = errors.New("download task canceled")
var ErrResourceUnavailable = errors.New("resource unavailable")

type RemoteResource[T any] struct {
	mu sync.Mutex
	wg sync.WaitGroup

	Name string

	logger    *CustomLogger
	scheduler *Scheduler

	Result *T

	coolingDown bool
	downloading bool

	DoDownload  func(ctx context.Context) (*T, error)
	CanDownload func() bool
}

func NewRemoteResource[T any](name string) *RemoteResource[T] {
	r := &RemoteResource[T]{
		logger:    NewLogger(),
		scheduler: NewScheduler(time.Second*3, time.Second),

		Name: name,

		DoDownload: func(ctx context.Context) (*T, error) {
			panic("Implementation required")
		},
		CanDownload: func() bool {
			return true
		},
	}
	r.wg.Add(1)

	return r
}

func NewJsonRemoteResource[T any](url string, clientFn func() *http.Client) *RemoteResource[T] {
	r := NewRemoteResource[T](url)
	r.DoDownload = func(ctx context.Context) (*T, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		r.logger.InfoLn("Downloading", url)

		resp, err := clientFn().Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 500 {
			r.scheduler.AddDelay(time.Second * 10)
			return nil, ErrResourceUnavailable
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			r.scheduler.Throttle()
			go func() {
				<-time.After(time.Minute)
				r.scheduler.ReleaseOneThrottle()
			}()
			return nil, errors.New("too many requests")
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to download, status: " + resp.Status)
		}

		r.scheduler.ResetDelay()

		var data T
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&data)
		if err != nil {
			return nil, err
		}

		return &data, nil
	}

	return r
}

func (r *RemoteResource[T]) StartDownload(ctx context.Context) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.CanDownload() || r.coolingDown || r.downloading {
		return false
	}

	r.downloading = true

	go func() {
		defer func() {
			r.downloading = false
		}()

		data, err := r.DoDownload(ctx)
		if err != nil {
			if errors.Is(err, ErrCanceled) {
				r.wg.Done()
			} else {
				if errors.Is(err, ErrResourceUnavailable) {
					r.logger.Errorf("Resource %s unavailable", r.Name)
				} else {
					r.logger.Errorf("Failed to download %s: %v", r.Name, err)
				}
				r.planNextRetry(ctx)
			}
		} else {
			r.Result = data
			r.wg.Done()
		}
	}()

	return true
}

func (r *RemoteResource[T]) planNextRetry(ctx context.Context) {
	r.coolingDown = true
	go func() {
		select {
		case <-ctx.Done():
			r.coolingDown = false
		case <-time.After(r.scheduler.ReserveWithDelay()):
			r.coolingDown = false
			r.StartDownload(ctx)
		}
	}()
}

func (r *RemoteResource[T]) Get() *T {
	r.wg.Wait()
	return r.Result
}
