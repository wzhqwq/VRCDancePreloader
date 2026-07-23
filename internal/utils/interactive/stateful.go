package interactive

import (
	"context"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type RunnerStatus struct {
	Running bool
	Error   error
}

type StatefulService interface {
	SubscribeStatus() *utils.EventSubscriber[RunnerStatus]
	Status() RunnerStatus
	Start()
	Stop()
	Restart()
}

type StatefulTester interface {
	SubscribeStatus() *utils.EventSubscriber[TesterStatus]
	Status() TesterStatus
	Test()
	Reset()
}

type StatefulSetting[T any] interface {
	Get() T
	Save(T) error
	Subscribe() *utils.EventSubscriber[T]
	SubscribeWhether(func(T) bool) *utils.EventSubscriber[bool]
}

type StatefulRemoteData[T any] interface {
	BindAvailability(availabilitySubFn AvailabilitySubFn)
	BindScheduler(scheduler *utils.Scheduler)
	BindRetry(retryPolicy *utils.RetryPolicy)

	Request()
	Release()
	Refresh()
	Invalidate()
	Snapshot() RemoteSnapshot[T]
	BlockedGet(ctx context.Context) (T, error)
	WaitValid(ctx context.Context) error
	Subscribe() *utils.EventSubscriber[RemoteSnapshot[T]]
	SubscribeStatus() *utils.EventSubscriber[RemoteStatus]
	Close()
}
