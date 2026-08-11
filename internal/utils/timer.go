package utils

import (
	"container/heap"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrManagerClosed = errors.New("countdown manager is closed")
	ErrSessionClosed = errors.New("countdown session is closed")
)

type CountdownManager struct {
	commands chan any
	done     chan struct{}

	closeOnce sync.Once
}

type CountdownSession struct {
	C <-chan time.Time

	manager *CountdownManager
	id      uint64

	c    chan time.Time
	done chan struct{}

	untilMu sync.RWMutex
	until   time.Time

	closed atomic.Bool
}

type sessionState struct {
	session *CountdownSession

	next time.Time

	index int
}

type sessionHeap []*sessionState

func (h sessionHeap) Len() int {
	return len(h)
}

func (h sessionHeap) Less(i, j int) bool {
	if h[i].next.Equal(h[j].next) {
		return h[i].session.id < h[j].session.id
	}
	return h[i].next.Before(h[j].next)
}

func (h sessionHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *sessionHeap) Push(value any) {
	state := value.(*sessionState)
	state.index = len(*h)
	*h = append(*h, state)
}

func (h *sessionHeap) Pop() any {
	old := *h
	n := len(old)

	state := old[n-1]
	old[n-1] = nil
	state.index = -1

	*h = old[:n-1]
	return state
}

type createSessionCommand struct {
	until time.Time
	reply chan createSessionResult
}

type createSessionResult struct {
	session *CountdownSession
	err     error
}

type updateSessionCommand struct {
	session *CountdownSession
	until   time.Time
	reply   chan error
}

type closeSessionCommand struct {
	session *CountdownSession
	reply   chan error
}

type closeManagerCommand struct {
	reply chan struct{}
}

func NewCountdownManager() *CountdownManager {
	manager := &CountdownManager{
		commands: make(chan any),
		done:     make(chan struct{}),
	}

	go manager.run()
	return manager
}

func (m *CountdownManager) NewSession(until time.Time) (*CountdownSession, error) {
	reply := make(chan createSessionResult, 1)

	command := createSessionCommand{
		until: until,
		reply: reply,
	}

	select {
	case m.commands <- command:
	case <-m.done:
		return nil, ErrManagerClosed
	}

	select {
	case result := <-reply:
		return result.session, result.err

	case <-m.done:
		return nil, ErrManagerClosed
	}
}

func (m *CountdownManager) Close() {
	m.closeOnce.Do(func() {
		reply := make(chan struct{}, 1)

		command := closeManagerCommand{
			reply: reply,
		}

		select {
		case m.commands <- command:
			select {
			case <-reply:
			case <-m.done:
			}

		case <-m.done:
		}
	})

	<-m.done
}

func (s *CountdownSession) SetUntil(until time.Time) error {
	if until == s.until {
		return nil
	}
	if s.closed.Load() {
		return ErrSessionClosed
	}

	reply := make(chan error, 1)

	command := updateSessionCommand{
		session: s,
		until:   until,
		reply:   reply,
	}

	select {
	case s.manager.commands <- command:

	case <-s.done:
		return ErrSessionClosed

	case <-s.manager.done:
		return ErrManagerClosed
	}

	select {
	case err := <-reply:
		return err

	case <-s.done:
		return ErrSessionClosed

	case <-s.manager.done:
		if s.closed.Load() {
			return ErrSessionClosed
		}
		return ErrManagerClosed
	}
}

func (s *CountdownSession) Until() time.Time {
	s.untilMu.RLock()
	until := s.until
	s.untilMu.RUnlock()

	return until
}

func (s *CountdownSession) Seconds() int {
	remaining := time.Until(s.Until())
	if remaining <= 0 {
		return 0
	}

	seconds := remaining / time.Second
	if remaining%time.Second != 0 {
		seconds++
	}

	maxInt := int64(^uint(0) >> 1)
	if int64(seconds) > maxInt {
		return int(maxInt)
	}

	return int(seconds)
}

func (s *CountdownSession) Close() error {
	if s.closed.Load() {
		return nil
	}

	reply := make(chan error, 1)

	command := closeSessionCommand{
		session: s,
		reply:   reply,
	}

	select {
	case s.manager.commands <- command:

	case <-s.done:
		return nil

	case <-s.manager.done:
		return nil
	}

	select {
	case err := <-reply:
		return err

	case <-s.done:
		return nil

	case <-s.manager.done:
		return nil
	}
}

func (s *CountdownSession) setUntil(until time.Time) {
	s.untilMu.Lock()
	s.until = until
	s.untilMu.Unlock()
}

func (m *CountdownManager) run() {
	states := make(map[uint64]*sessionState)

	queue := make(sessionHeap, 0)
	heap.Init(&queue)

	var nextSessionID uint64

	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}

	var timerC <-chan time.Time

	stopTimer := func() {
		if timerC == nil {
			return
		}

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		timerC = nil
	}

	armTimer := func() {
		stopTimer()

		if queue.Len() == 0 {
			return
		}

		delay := time.Until(queue[0].next)
		if delay < 0 {
			delay = 0
		}

		timer.Reset(delay)
		timerC = timer.C
	}

	removeFromQueue := func(state *sessionState) {
		if state.index >= 0 {
			heap.Remove(&queue, state.index)
		}
	}

	scheduleAtOrAfterNow := func(
		state *sessionState,
		now time.Time,
	) {
		removeFromQueue(state)

		next, ok := firstBoundaryAtOrAfter(
			now,
			state.session.Until(),
		)
		if !ok {
			return
		}

		state.next = next
		heap.Push(&queue, state)
	}

	closeSession := func(state *sessionState) {
		removeFromQueue(state)
		delete(states, state.session.id)

		if state.session.closed.CompareAndSwap(false, true) {
			close(state.session.done)
			close(state.session.c)
		}
	}

	for {
		select {
		case rawCommand := <-m.commands:
			switch command := rawCommand.(type) {
			case createSessionCommand:
				nextSessionID++

				channel := make(chan time.Time, 1)

				session := &CountdownSession{
					C:       channel,
					manager: m,
					id:      nextSessionID,
					c:       channel,
					done:    make(chan struct{}),
					until:   command.until,
				}

				state := &sessionState{
					session: session,
					index:   -1,
				}

				states[session.id] = state

				scheduleAtOrAfterNow(state, time.Now())

				command.reply <- createSessionResult{
					session: session,
				}

			case updateSessionCommand:
				state, exists := states[command.session.id]

				if !exists ||
					state.session != command.session ||
					command.session.closed.Load() {
					command.reply <- ErrSessionClosed
					break
				}

				command.session.setUntil(command.until)
				scheduleAtOrAfterNow(state, time.Now())

				command.reply <- nil

			case closeSessionCommand:
				state, exists := states[command.session.id]

				if exists && state.session == command.session {
					closeSession(state)
				}

				command.reply <- nil

			case closeManagerCommand:
				stopTimer()

				for _, state := range states {
					closeSession(state)
				}

				close(m.done)
				command.reply <- struct{}{}
				return
			}

			armTimer()

		case <-timerC:
			timerC = nil
			now := time.Now()

			for queue.Len() > 0 &&
				!queue[0].next.After(now) {
				state := heap.Pop(&queue).(*sessionState)
				scheduledAt := state.next

				select {
				case state.session.c <- scheduledAt:
				default:
				}

				next, ok := firstBoundaryAfter(now, state.session.Until())
				if ok {
					state.next = next
					heap.Push(&queue, state)
				}
			}

			armTimer()
		}
	}
}

func firstBoundaryAtOrAfter(now time.Time, until time.Time) (time.Time, bool) {
	if now.After(until) {
		return time.Time{}, false
	}

	remaining := until.Sub(now)
	whole := (remaining / time.Second) * time.Second

	candidate := until.Add(-whole)

	if candidate.Before(now) {
		candidate = candidate.Add(time.Second)
	}

	if candidate.After(until) {
		return time.Time{}, false
	}

	return candidate, true
}

func firstBoundaryAfter(now time.Time, until time.Time) (time.Time, bool) {
	if !now.Before(until) {
		return time.Time{}, false
	}

	remaining := until.Sub(now)
	whole := (remaining / time.Second) * time.Second

	candidate := until.Add(-whole)

	if !candidate.After(now) {
		candidate = candidate.Add(time.Second)
	}

	if candidate.After(until) {
		return time.Time{}, false
	}

	return candidate, true
}
