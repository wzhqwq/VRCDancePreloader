package preloader

import (
	"context"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func (s *Service) makeSureDownloading(item *song.StatefulSong) {
	if !item.StateMachine().CanStartDownload() {
		return
	}

	id := item.SongId()
	sm := item.StateMachine()
	// check whether it have bound session
	if !sm.BindCache(func() (types.CDNFileSession, error) {
		return s.cacheSvc.CreateSession("video")
	}) {
		return
	}

	sm.BindTask(func(songSession types.CDNFileSession) *downloader.ManagedTask {
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
	})
}

func (s *Service) getResource(item *song.StatefulSong, ctx context.Context) (types.CDNResource, error) {
	// get logger from item
	logger := item.ValidLogger(ctx)

	session, err := s.cacheSvc.CreateSession("video")
	if err != nil {
		return nil, err
	}

	err = session.Open(item.SongId(), logger)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		session.Close(logger)
	}()

	// finally, return the resource
	return session, nil
}
