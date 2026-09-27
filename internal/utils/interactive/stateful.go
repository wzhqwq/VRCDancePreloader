package interactive

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type RunnerStatus struct {
	Running bool
	Error   error
}

// StatefulService and StatefulTester expose their status both as a value
// (Status) and as a level (WatchStatus): the watcher is only a wake-up, so a
// consumer that is woken reads Status and cannot act on a status that has already
// been replaced. Each caller gets its own watcher and closes it when it stops.
type StatefulService interface {
	Status() RunnerStatus
	WatchStatus() *utils.LevelWatcher[RunnerStatus]
	Start()
	Stop()
	Restart()
}

type StatefulTester interface {
	Status() TesterStatus
	WatchStatus() *utils.LevelWatcher[TesterStatus]
	Test()
	Reset()
}

// StatefulSetting exposes a configuration value both as a value (Get) and as a
// level (Watch): a consumer is woken and reads Get, so it can never render a
// value that has already been replaced. Each caller gets its own watcher and
// closes it when it stops.
//
// There is deliberately no SubscribeWhether: "tell me whether the predicate holds
// after each change" is the consumer's own `if pred(setting.Get())` after a
// wake-up.
type StatefulSetting[T any] interface {
	Get() T
	Save(T) error
	Watch() *utils.LevelWatcher[T]
}

// StatefulRemoteData[T] used to live here: an interface with no implementer that
// also could not be implemented (its BindRetry took *utils.RetryPolicy while
// RemoteManager.BindRetry takes a value, and it mixed methods of
// *RemoteManager[T] with methods of *RemoteHandle[T]). It was deleted on
// 2026-09-25 by the maintainer's ruling; the content is in git history and in
// notes/19 if the contract ever has to be drafted again.
