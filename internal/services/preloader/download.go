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
const entryInitWaitTimeout = 30 * time.Second

func (s *Service) cacheBinder() (types.CDNFileSession, error) {
	return s.cacheSvc.CreateSession("video")
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

	snap := handle.Snapshot()

	ch := handle.Subscribe()
	defer ch.Close()

	for {
		if snap.Status.Valid() {
			break
		}

		select {
		case <-stopCh:
			return api.DuDuOriginalVideoInfo{}, false
		case snap = <-ch.Channel:
		}
	}

	return snap.Data, true
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

	// wait for video info
	err = third_parties.GetProviderById(id).ResolvedVideo(id).WaitValid(ctx)
	if err != nil {
		return nil, err
	}

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
