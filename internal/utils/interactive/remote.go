package interactive

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var (
	ErrTemporarilyUnavailable = errors.New("")
	ErrUnrecoverable          = errors.New("")
)

type RemotePhase int

const (
	RemoteIdle RemotePhase = iota
	RemoteLoading
	RemoteReady
	RemoteRefreshing
	RemoteError
	RemoteErrorRetrying
)

type RemoteStatus struct {
	Phase       RemotePhase
	Placeholder bool
	Err         error

	CooldownUntil time.Time
	RetryAttempts string
}

func (s RemoteStatus) Fetching() bool {
	return s.Phase == RemoteLoading || s.Phase == RemoteRefreshing
}

func (s RemoteStatus) Valid() bool {
	return s.Phase == RemoteReady
}

func (s RemoteStatus) String() string {
	var result string
	if s.Phase == RemoteErrorRetrying {
		result = i18n.T("status_remote_"+strconv.Itoa(int(s.Phase)), goeasyi18n.Options{
			Data: map[string]interface{}{
				"Attempts": s.RetryAttempts,
			},
		})
	} else {
		result = i18n.T("status_remote_" + strconv.Itoa(int(s.Phase)))
	}
	if !s.CooldownUntil.IsZero() {
		result = result + i18n.T("status_remote_cooling", goeasyi18n.Options{
			Data: map[string]interface{}{
				"Seconds": int(s.CooldownUntil.Sub(time.Now()).Seconds()),
			},
		})
	}
	return result
}

func (s RemoteStatus) Color() fyne.ThemeColorName {
	switch s.Phase {
	case RemoteIdle:
		return theme.ColorNamePlaceHolder
	case RemoteLoading, RemoteRefreshing:
		return theme.ColorNamePrimary
	case RemoteReady:
		return theme.ColorNameSuccess
	default:
		return theme.ColorNameError
	}
}

type RemoteSnapshot[T any] struct {
	Data    T
	HasData bool
	Status  RemoteStatus
}

type fetchRequest[T any] struct {
	entry      *remoteEntry[T]
	generation uint64
	refresh    bool
}

type GetFn[T any] func(id string, ctx context.Context) (T, error)

type AvailabilitySubFn func() *utils.EventSubscriber[bool]

type RemoteManager[T any] struct {
	mu sync.RWMutex

	entries map[string]*remoteEntry[T]

	getFn         GetFn[T]
	placeholderFn func(id string) T

	scheduler   *utils.Scheduler
	retryPolicy utils.RetryPolicy
	logger      utils.LoggerImpl

	queue chan fetchRequest[T]
	wg    sync.WaitGroup

	maxCacheEntries int

	availabilitySubFn AvailabilitySubFn

	closeOnce sync.Once
	closeCh   chan struct{}
	closed    atomic.Bool
}

func NewRemoteManager[T any](
	getFn GetFn[T],
	placeholderFn func(id string) T,
	maxCacheEntries int,
	maxParallelism int,
) *RemoteManager[T] {
	m := &RemoteManager[T]{
		entries: make(map[string]*remoteEntry[T]),

		getFn:         getFn,
		placeholderFn: placeholderFn,

		queue: make(chan fetchRequest[T], 64),

		maxCacheEntries: maxCacheEntries,

		closeCh: make(chan struct{}),
	}
	for i := 0; i < maxParallelism; i++ {
		m.wg.Go(m.worker)
	}
	return m
}

func (m *RemoteManager[T]) BindAvailability(availabilitySubFn AvailabilitySubFn) {
	m.availabilitySubFn = availabilitySubFn
	go m.availabilityLoop()
}

func (m *RemoteManager[T]) BindScheduler(scheduler *utils.Scheduler) {
	m.scheduler = scheduler
}

func (m *RemoteManager[T]) BindRetry(retryPolicy utils.RetryPolicy) {
	m.retryPolicy = retryPolicy
}

func (m *RemoteManager[T]) BindLogger(logger utils.LoggerImpl) {
	m.logger = logger
}

func (m *RemoteManager[T]) enqueue(
	entry *remoteEntry[T],
	refresh bool,
) {
	entry.mu.Lock()

	if entry.queued {
		entry.mu.Unlock()
		return
	}

	if entry.fetching && !refresh {
		entry.mu.Unlock()
		return
	}

	entry.generation++
	generation := entry.generation
	entry.queued = true

	entry.mu.Unlock()

	req := fetchRequest[T]{
		entry:      entry,
		generation: generation,
		refresh:    refresh,
	}

	select {
	case m.queue <- req:
	case <-m.closeCh:
	}
}

func (m *RemoteManager[T]) worker() {
	for {
		select {
		case req := <-m.queue:
			m.execute(req)

		case <-m.closeCh:
			return
		}
	}
}

func waitDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *RemoteManager[T]) execute(req fetchRequest[T]) {
	entry := req.entry

	ctx, cancel := context.WithCancel(context.Background())

	entry.mu.Lock()
	if req.generation != entry.generation {
		entry.mu.Unlock()
		cancel()
		return
	}

	entry.cancel = cancel
	entry.fetching = true
	entry.queued = false
	entry.runningGeneration = req.generation

	if entry.hasData && !entry.placeholder {
		entry.status = RemoteStatus{
			Phase: RemoteRefreshing,
		}
	} else {
		entry.status = RemoteStatus{
			Phase:       RemoteLoading,
			Placeholder: entry.placeholder,
		}
	}

	entry.mu.Unlock()

	data, err := utils.Retry[T](
		ctx,
		m.retryPolicy,
		func(ctx context.Context, attempt int) (data T, canRetry bool, err error) {
			var delay time.Duration
			entry.mu.Lock()
			if m.scheduler != nil {
				delay = m.scheduler.Reserve()
			}
			if delay > 0 {
				entry.status.CooldownUntil = time.Now().Add(delay)
			} else {
				entry.status.CooldownUntil = time.Time{}
			}
			snapshot := entry.snapshotLocked()
			entry.mu.Unlock()

			entry.em.NotifySubscribers(snapshot)

			err = waitDelay(ctx, delay)
			if err == nil {
				data, err = m.getFn(entry.id, ctx)
			}

			var throttleErr *utils.ThrottledError
			canRetry = true

			switch {
			case errors.Is(err, context.Canceled):
				if m.logger != nil {
					m.logger.InfoLn("Canceled", entry.id)
				}
				canRetry = false
				return
			case errors.Is(err, ErrTemporarilyUnavailable):
				if m.scheduler != nil {
					if m.logger != nil {
						m.logger.WarnLn("Failed to fetch", entry.id, "and will retry after 30 seconds:", err)
					}
					m.scheduler.Pause(time.Second * 30)
				} else {
					canRetry = false
				}
			case errors.As(err, &throttleErr):
				if m.scheduler != nil {
					if m.logger != nil {
						m.logger.WarnLn("Fetching", entry.id, "fails and triggers a throttle control")
					}
					m.scheduler.Throttle(throttleErr.RetryAfter)
				} else {
					canRetry = false
				}
			case errors.Is(err, ErrUnrecoverable):
				canRetry = false
			}

			if !canRetry {
				if m.logger != nil {
					m.logger.ErrorLn("Failed to fetch", entry.id, "and will retry when it becomes available:", err)
				}
				return
			}

			if attempt < m.retryPolicy.MaxRetries {
				entry.mu.Lock()
				entry.status.Phase = RemoteErrorRetrying
				entry.status.Err = err
				entry.status.RetryAttempts = fmt.Sprintf("%d / %d", attempt+1, m.retryPolicy.MaxRetries)
				snapshot = entry.snapshotLocked()
				entry.mu.Unlock()
			}

			entry.em.NotifySubscribers(snapshot)

			return
		},
	)

	m.finishFetch(entry, req.generation, data, err)
}

func (m *RemoteManager[T]) finishFetch(
	entry *remoteEntry[T],
	generation uint64,
	data T,
	err error,
) {
	entry.mu.Lock()

	if generation != entry.runningGeneration {
		entry.mu.Unlock()
		return
	}

	entry.runningGeneration = 0
	entry.cancel = nil
	entry.fetching = false

	if generation != entry.generation {
		snapshot := entry.snapshotLocked()
		entry.mu.Unlock()

		entry.em.NotifySubscribers(snapshot)
		m.evictIfNeeded()
		return
	}

	switch {
	case err == nil:
		entry.data = data
		entry.hasData = true
		entry.placeholder = false
		entry.status = RemoteStatus{
			Phase: RemoteReady,
		}

	case errors.Is(err, context.Canceled):
		if entry.hasData && !entry.placeholder {
			entry.status = RemoteStatus{
				Phase: RemoteReady,
			}
		} else {
			entry.status = RemoteStatus{
				Phase:       RemoteIdle,
				Placeholder: entry.placeholder,
			}
		}

	default:
		if entry.hasData && !entry.placeholder {
			entry.status = RemoteStatus{
				Phase: RemoteReady,
				Err:   err,
			}
		} else {
			entry.status = RemoteStatus{
				Phase:       RemoteError,
				Placeholder: entry.placeholder,
				Err:         err,
			}
		}
	}

	entry.lastAccess = time.Now()
	snapshot := entry.snapshotLocked()

	entry.mu.Unlock()

	entry.em.NotifySubscribers(snapshot)
	m.evictIfNeeded()
}

func (m *RemoteManager[T]) availabilityLoop() {
	sub := m.availabilitySubFn()
	if sub == nil {
		return
	}
	defer sub.Close()

	for {
		select {
		case available, ok := <-sub.Channel:
			if !ok {
				return
			}

			if available {
				m.retryUnavailableEntries()
			}

		case <-m.closeCh:
			return
		}
	}
}

func (m *RemoteManager[T]) retryUnavailableEntries() {
	m.mu.RLock()
	entries := make([]*remoteEntry[T], 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}
	m.mu.RUnlock()

	for _, entry := range entries {
		entry.mu.RLock()
		shouldRetry :=
			entry.refCount.Load() > 0 &&
				!entry.fetching &&
				!entry.queued &&
				(entry.status.Phase == RemoteError ||
					!entry.hasData ||
					entry.placeholder)
		entry.mu.RUnlock()

		if shouldRetry {
			m.enqueue(entry, false)
		}
	}
}

func (m *RemoteManager[T]) evictIfNeeded() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for len(m.entries) > m.maxCacheEntries {
		var victim *remoteEntry[T]

		for _, entry := range m.entries {
			entry.mu.RLock()
			evictable :=
				entry.refCount.Load() == 0 &&
					!entry.fetching &&
					!entry.queued
			lastAccess := entry.lastAccess
			entry.mu.RUnlock()

			if !evictable {
				continue
			}

			if victim == nil || lastAccess.Before(victim.lastAccess) {
				victim = entry
			}
		}

		if victim == nil {
			return
		}

		delete(m.entries, victim.id)
	}
}

func (m *RemoteManager[T]) release(entry *remoteEntry[T]) {
	if entry == nil {
		return
	}

	n := entry.refCount.Add(-1)
	if n < 0 {
		entry.refCount.Add(1)
		panic("RemoteManager.release: negative reference count")
	}

	entry.mu.Lock()
	entry.lastAccess = time.Now()
	entry.mu.Unlock()

	if n != 0 {
		return
	}

	go m.cancelIfUnusedAfter(entry, 3*time.Second)
}

func (m *RemoteManager[T]) cancelIfUnusedAfter(
	entry *remoteEntry[T],
	delay time.Duration,
) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-m.closeCh:
		return
	}

	if entry.refCount.Load() != 0 {
		return
	}

	var cancel context.CancelFunc

	entry.mu.Lock()

	if entry.refCount.Load() == 0 && entry.fetching {
		cancel = entry.cancel
	}

	entry.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	m.evictIfNeeded()
}

func (m *RemoteManager[T]) getOrCreate(id string) *remoteEntry[T] {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry := m.entries[id]; entry != nil {
		return entry
	}

	entry := &remoteEntry[T]{
		id:         id,
		lastAccess: time.Now(),
		em:         utils.NewEventManager[RemoteSnapshot[T]](),
		status: RemoteStatus{
			Phase: RemoteIdle,
		},
	}

	if m.placeholderFn != nil {
		entry.data = m.placeholderFn(id)
		entry.hasData = true
		entry.placeholder = true
		entry.status.Placeholder = true
	}

	m.entries[id] = entry
	return entry
}

func (m *RemoteManager[T]) invalidateEntry(entry *remoteEntry[T]) {
	if entry == nil || m.closed.Load() {
		return
	}

	var (
		cancel      context.CancelFunc
		snapshot    RemoteSnapshot[T]
		shouldFetch bool
	)

	entry.mu.Lock()

	entry.lastAccess = time.Now()

	entry.generation++

	cancel = entry.cancel

	entry.queued = false

	if entry.hasData && !entry.placeholder {
		var zero T
		entry.data = zero
		entry.hasData = false

		if m.placeholderFn != nil {
			entry.data = m.placeholderFn(entry.id)
			entry.hasData = true
			entry.placeholder = true
			entry.status = RemoteStatus{
				Phase:       RemoteIdle,
				Placeholder: true,
			}
		} else {
			entry.placeholder = false
			entry.status = RemoteStatus{
				Phase: RemoteIdle,
			}
		}
	} else {
		entry.status = RemoteStatus{
			Phase:       RemoteIdle,
			Placeholder: entry.placeholder,
		}
	}

	shouldFetch = entry.refCount.Load() > 0
	snapshot = entry.snapshotLocked()

	entry.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	entry.em.NotifySubscribers(snapshot)

	if shouldFetch {
		m.enqueue(entry, false)
	} else {
		m.evictIfNeeded()
	}
}

func (m *RemoteManager[T]) Acquire(id string) *RemoteHandle[T] {
	if id == "" {
		panic("RemoteManager.Acquire: empty id")
	}
	if m.closed.Load() {
		return nil
	}

	var (
		entry      *remoteEntry[T]
		created    bool
		shouldLoad bool
	)

	m.mu.Lock()

	if m.closed.Load() {
		m.mu.Unlock()
		return nil
	}

	entry = m.entries[id]
	if entry == nil {
		entry = &remoteEntry[T]{
			id:         id,
			lastAccess: time.Now(),
			em:         utils.NewEventManager[RemoteSnapshot[T]](),
			status: RemoteStatus{
				Phase: RemoteIdle,
			},
		}

		if m.placeholderFn != nil {
			entry.data = m.placeholderFn(id)
			entry.hasData = true
			entry.placeholder = true
			entry.status.Placeholder = true
		}

		m.entries[id] = entry
		created = true
	}

	m.mu.Unlock()

	n := entry.refCount.Add(1)

	entry.mu.Lock()
	entry.lastAccess = time.Now()

	if n == 1 {
		shouldLoad =
			!entry.fetching &&
				!entry.queued &&
				(!entry.hasData ||
					entry.placeholder ||
					entry.status.Phase == RemoteIdle ||
					entry.status.Phase == RemoteError)
	}

	entry.mu.Unlock()

	if shouldLoad {
		m.enqueue(entry, false)
	}

	if created {
		m.evictIfNeeded()
	}

	return &RemoteHandle[T]{
		manager: m,
		entry:   entry,
	}
}

func (m *RemoteManager[T]) Close() {
	m.closeOnce.Do(func() {
		m.closed.Store(true)
		close(m.closeCh)

		m.mu.RLock()
		entries := make([]*remoteEntry[T], 0, len(m.entries))
		for _, entry := range m.entries {
			entries = append(entries, entry)
		}
		m.mu.RUnlock()

		for _, entry := range entries {
			entry.mu.Lock()
			cancel := entry.cancel
			entry.cancel = nil
			entry.mu.Unlock()

			if cancel != nil {
				cancel()
			}
		}

		m.wg.Wait()
	})
}

func (m *RemoteManager[T]) Refresh(id string) {
	entry := m.entries[id]
	if entry != nil {
		m.enqueue(entry, true)
	}
}

func (m *RemoteManager[T]) RefreshActive() {
	m.mu.RLock()
	entries := make([]*remoteEntry[T], 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}
	m.mu.RUnlock()

	for _, entry := range entries {
		if entry.refCount.Load() > 0 {
			m.enqueue(entry, true)
		} else {
			m.invalidateEntry(entry)
		}
	}
}

func (m *RemoteManager[T]) Invalidate(id string) {
	entry := m.entries[id]
	if entry != nil {
		m.invalidateEntry(entry)
	}
}

//func (m *RemoteManager[T]) InvalidateAll() {
//
//}

func (m *RemoteManager[T]) ModifyPlaceholder(id string, data T) bool {
	return m.ModifyPlaceholderFn(id, func(_ T) T {
		return data
	})
}

func (m *RemoteManager[T]) ModifyPlaceholderFn(id string, modify func(current T) T) bool {
	if id == "" || modify == nil {
		return false
	}

	var (
		entry    *remoteEntry[T]
		snapshot RemoteSnapshot[T]
	)

	m.mu.RLock()

	if m.closed.Load() {
		m.mu.RUnlock()
		return false
	}

	entry = m.entries[id]
	if entry == nil {
		m.mu.RUnlock()
		return false
	}

	entry.mu.Lock()

	if entry.hasData {
		if !entry.placeholder {
			entry.mu.Unlock()
			m.mu.RUnlock()
			return false
		}
	} else {
		entry.hasData = true
		entry.placeholder = true
	}

	entry.data = modify(entry.data)
	entry.lastAccess = time.Now()
	entry.status.Placeholder = true

	snapshot = entry.snapshotLocked()

	entry.mu.Unlock()
	m.mu.RUnlock()

	entry.em.NotifySubscribers(snapshot)
	return true
}

type remoteEntry[T any] struct {
	mu sync.RWMutex

	id string

	data        T
	hasData     bool
	placeholder bool
	status      RemoteStatus

	refCount atomic.Int64

	generation        uint64
	runningGeneration uint64

	queued   bool
	fetching bool

	cancel context.CancelFunc

	lastAccess time.Time

	em *utils.EventManager[RemoteSnapshot[T]]
}

func (e *remoteEntry[T]) snapshotLocked() RemoteSnapshot[T] {
	return RemoteSnapshot[T]{
		Data:    e.data,
		HasData: e.hasData,
		Status:  e.status,
	}
}

func (e *remoteEntry[T]) snapshot() RemoteSnapshot[T] {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.snapshotLocked()
}

type RemoteHandle[T any] struct {
	manager *RemoteManager[T]
	entry   *remoteEntry[T]

	released atomic.Bool
}

func (h *RemoteHandle[T]) Snapshot() RemoteSnapshot[T] {
	return h.entry.snapshot()
}

func (h *RemoteHandle[T]) WaitValid(ctx context.Context) error {
	ch := h.Subscribe()
	defer ch.Close()

	snapshot := h.Snapshot()
	if snapshot.Status.Valid() {
		return nil
	}

	for {
		select {
		case <-h.manager.closeCh:
			return errors.New("closed")
		case <-ctx.Done():
			return ctx.Err()
		case snapshot = <-ch.Channel:
			if snapshot.Status.Valid() {
				return nil
			}
		}
	}
}

func (h *RemoteHandle[T]) Refresh() {
	if h == nil || h.manager == nil || h.entry == nil {
		return
	}
	if h.released.Load() {
		return
	}

	h.manager.enqueue(h.entry, true)
}

func (h *RemoteHandle[T]) Subscribe() *utils.EventSubscriber[RemoteSnapshot[T]] {
	return h.entry.em.SubscribeEvent()
}

func (h *RemoteHandle[T]) SubscribeStatus() *utils.EventSubscriber[RemoteStatus] {
	return utils.PipeEvent(h.entry.em, func(snap RemoteSnapshot[T]) (RemoteStatus, bool) {
		return snap.Status, true
	})
}

func (h *RemoteHandle[T]) Release() {
	if h == nil || h.manager == nil || h.entry == nil {
		return
	}

	if h.released.CompareAndSwap(false, true) {
		h.manager.release(h.entry)
	}
}

func (h *RemoteHandle[T]) BlockedGet(ctx context.Context) (T, error) {
	ch := h.Subscribe()
	defer ch.Close()

	snapshot := h.Snapshot()
	if snapshot.HasData {
		return snapshot.Data, nil
	}

	for {
		select {
		case <-h.manager.closeCh:
			return snapshot.Data, errors.New("closed")
		case <-ctx.Done():
			return snapshot.Data, ctx.Err()
		case snapshot = <-ch.Channel:
			if snapshot.HasData {
				return snapshot.Data, nil
			}
		}
	}
}
