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

func (s *Service) makeSureDownloading(item *song.StatefulSong) {
	sm := item.StateMachine()
	if sm.BindCache(s.cacheBinder) {
		sm.BindTask(s.taskBinder)
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

	// finally, return the resource
	return session, nil
}
