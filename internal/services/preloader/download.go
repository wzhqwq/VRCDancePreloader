package preloader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

// entryInitWaitTimeout bounds how long a video request waits for its cache
// entry to consume the resolved remote info. The download task normally does
// that within a few seconds; if it has not happened by then something is wrong,
// and failing the request is preferable to holding the proxy connection open
// indefinitely.
//
// This applies to videos that are supposed to be downloaded. A request the
// queue is not going to serve yet never reaches this wait at all: it is served
// from the origin instead (see Service.requestTemporary).
const entryInitWaitTimeout = 30 * time.Second

// entryReadyWatcherTimeout bounds the lifetime of the goroutine that watches a
// cache entry for a request that is being served from the origin.
//
// The entry is expected to become usable within seconds, and the watch ends
// then. The timeout only bounds the other direction: a client that keeps a
// pass-through connection wide open on a video whose download never happens
// would otherwise keep the goroutine alive for the whole connection. It only
// watches, so giving up costs nothing but the opportunity to hand over.
const entryReadyWatcherTimeout = 10 * time.Minute

func (s *Service) cacheBinder() (types.CDNFileSession, error) {
	return s.cacheSvc.CreateSession("video")
}

// hasCompleteCache reports whether the video is already fully cached, and so can
// be served without waiting for anything.
//
// "Fully" rather than "has a header": a partially downloaded file opens and
// initializes instantly, so it cannot be told apart from a complete one by
// getResource alone, and serving it would be an unnecessary pass-through — the
// entry is ready, and the reader fills the gaps as they arrive.
func (s *Service) hasCompleteCache(id string, ctx context.Context) bool {
	logger := utils.NewLogger("Cache Probe")

	session, err := s.cacheSvc.CreateSession("video")
	if err != nil {
		return false
	}
	defer session.Close()

	if err := session.Open(id, logger); err != nil {
		return false
	}

	file, err := session.AcquireFile()
	if err != nil {
		return false
	}
	complete := file.IsComplete()
	session.ReleaseFile()

	return complete
}

// WatchEntryReady reports, by closing the returned channel, when the cache entry
// of a video has become usable — the same condition getResource waits for before
// it serves the video from the cache.
//
// It is how a pass-through response learns that it should stop: the entry is
// initialized exactly when serving from the cache becomes possible, and the
// download task is what initializes it.
//
// The channel is closed at most once and the goroutine always terminates: it
// stops when the request is over, when the watch times out, or when the entry is
// ready.
func (s *Service) WatchEntryReady(id string, ctx context.Context) <-chan struct{} {
	ready := make(chan struct{})

	go func() {
		logger := utils.NewLogger("Cache Watch")

		// The watch is tied to the request only for cancellation; the deadline is
		// its own, so that a request that stays open for a long time still
		// releases it eventually.
		watchCtx, cancelWatch := context.WithTimeout(context.WithoutCancel(ctx), entryReadyWatcherTimeout)
		defer cancelWatch()

		session, err := s.cacheSvc.CreateSession("video")
		if err != nil {
			logger.ErrorLn("Cannot watch the cache entry of", id, ":", err)
			return
		}
		defer session.Close()

		if err := session.Open(id, logger); err != nil {
			logger.ErrorLn("Cannot watch the cache entry of", id, ":", err)
			return
		}

		if err := session.WaitInitialized(watchCtx); err != nil {
			logger.InfoLn("Stopped watching the cache entry of", id, ":", err)
			return
		}

		logger.InfoLn("Cache entry of", id, "is initialized")
		close(ready)
	}()

	return ready
}

func (s *Service) taskBinder(songSession types.CDNFileSession, id string) *downloader.ManagedTask {
	return s.downloaderSvc.Download(
		id,
		func() task.RemoteProvider {
			return task.NewRWFileRemoteProvider(id, third_parties.GetProviderById(id).ResolvedVideo, songSession)
		},
		func(logger utils.LoggerImpl) task.LocalProvider {
			fileSession, err := s.cacheSvc.CreateSession("video")
			if err != nil {
				logger.ErrorLn("Failed to create session", err)
				return nil
			}
			return task.NewRWFileProvider(id, fileSession, logger)
		},
	)
}

func (s *Service) youtubeFallbackAvailable(id string) bool {
	if strings.HasPrefix(id, internal_id.PyPyInternalPrefix) {
		return lo.Contains(s.Cfg.UseYoutubeFallback, PyPyDanceRoomName)
	}
	if strings.HasPrefix(id, internal_id.DuDuInternalPrefix) {
		return lo.Contains(s.Cfg.UseYoutubeFallback, DuDuFitDanceRoomName)
	}
	return false
}

func (s *Service) duduFramerateFallbackAvailable() bool {
	return lo.Contains(s.Cfg.UseYoutubeFallback, DuDuFitDanceRoomName) && s.Cfg.HighFramerateFallback
}

func (s *Service) getDuDuOriginalInfo(id string, stopCh <-chan struct{}) (api.DuDuOriginalVideoInfo, bool) {
	handle := s.duduOriginalVideoInfoProvider.Info(id)
	defer handle.Release()

	ch := handle.Subscribe()
	defer ch.Close()

	for {
		// A notification is only a wake-up: the handle is what says whether the
		// info is usable, so a stale payload cannot make this return early (nor
		// hand out an older snapshot).
		snap := handle.Snapshot()
		if snap.Status.Valid() {
			return snap.Data, true
		}

		select {
		case <-stopCh:
			return api.DuDuOriginalVideoInfo{}, false
		case <-ch.Channel:
		}
	}
}

func (s *Service) bind(sm *song.StateMachine) {
	if sm.BindCache(s.cacheBinder) {
		sm.BindTask(s.taskBinder)
	}
}

func (s *Service) makeSureDownloading(item *song.StatefulSong) {
	sm := item.StateMachine()
	s.bind(sm)

	currentId := sm.CurrentSongId()

	if s.youtubeFallbackAvailable(currentId) {
		if strings.HasPrefix(currentId, internal_id.DuDuInternalPrefix) && s.duduFramerateFallbackAvailable() {
			s.Go(func(stopCh <-chan struct{}) {
				if info, ok := s.getDuDuOriginalInfo(currentId, stopCh); ok && info.Framerate > 40 {
					s.L().InfoLn("The framerate of", currentId, "is", info.Framerate, ", which is too high")
					if ytId, ok := internal_id.CheckYoutubeURL(info.OriginalURL); ok {
						s.L().InfoLn("Use YouTube fallback", ytId)
						if sm.Reset(internal_id.YtInternalPrefix + ytId) {
							s.bind(sm)
						}
					}
				}
			})
		}
	}
}

func (s *Service) getResource(item *song.StatefulSong, ctx context.Context) (types.CDNResource, error) {
	// get logger from item
	logger := item.ValidLogger(ctx)
	id := item.SongId()

	// access the cache
	session, err := s.cacheSvc.CreateSession("video")
	if err != nil {
		return nil, err
	}

	err = session.Open(id, logger)
	if err != nil {
		return nil, err
	}

	// close when context closes
	go func() {
		<-ctx.Done()
		session.Close(logger)
	}()

	// Having the resolved info available is not enough: the cache entry must
	// also have consumed it, because entry.GetResource reports a download
	// failure as long as the file has no total length. Only the download task
	// calls ReconcileRemoteInfo, so without this wait the request routinely
	// loses the race and the video is reported as broken.
	waitCtx, waitCancel := context.WithTimeout(ctx, entryInitWaitTimeout)
	defer waitCancel()

	err = session.WaitInitialized(waitCtx)
	if err != nil {
		return nil, fmt.Errorf("cache entry of %s is not initialized yet: %w", id, err)
	}

	// finally, return the resource
	return session, nil
}
