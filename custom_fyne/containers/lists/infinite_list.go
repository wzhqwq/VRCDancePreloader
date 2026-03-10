package lists

import (
	"sync/atomic"
	"time"
)

type InfiniteList[T DataWithID] struct {
	BaseList[T]

	nextPageLoading atomic.Bool
}

func NewInfiniteList[T DataWithID]() *InfiniteList[T] {
	l := &InfiniteList[T]{}
	l.Lazy = true
	l.scrolledFn = l.scrolled
	l.ExtendBaseList(l)
	return l
}

func (l *InfiniteList[T]) scrolled(offset float32) {
	closeToEnd := l.scrollHeight-offset < 40
	if closeToEnd {
		if l.nextPageLoading.CompareAndSwap(false, true) {
			go func() {
				l.appendItems()
				// waiting for rendering
				<-time.After(time.Millisecond * 100)
				l.nextPageLoading.Store(false)
			}()
		}
	}
}
