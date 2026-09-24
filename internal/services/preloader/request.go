package preloader

import (
	"context"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

// ErrPassThrough means "we are not going to serve this video from the cache
// yet, so the caller has to fetch it from the origin itself".
//
// It is a choice rather than a failure, which is why it is a sentinel error
// instead of a boolean: the handler has to be able to tell it apart from a real
// failure and log the two differently, and it has to tell it apart from a
// successful cache hit, which is also "not a failure".
//
// It is only ever returned for a video that is not in the playlist. Those
// requests come from the random-play queue, which the software cannot read, so
// they are legitimate but must not overtake the song that is playing now: the
// download is created at the tail of the queue and the client is served from the
// origin until that download has made the cache entry usable.
var ErrPassThrough = errors.New("serve this request from the origin directly")

func (s *Service) Request(id string, ctx context.Context) (types.CDNResource, error) {
	s.plMu.RLock()
	defer s.plMu.RUnlock()

	if s.pl == nil {
		return nil, errors.New("no valid playlist")
	}

	item := s.pl.SearchByInternalId(id)
	if item == nil {
		return s.requestTemporary(id, ctx)
	}

	s.makeSureDownloading(item)
	s.prioritizeForRequest(item.SongId())

	return s.getResource(item, ctx)
}

// prioritizeForRequest orders the queue for a request that is in the playlist.
//
// A request is not a reason to jump the whole queue: the video the client asks
// for is the one it is about to play, so it belongs at the front — but only
// *after* the song that is playing right now, which is the one whose download
// the user is actually waiting on. Naming both ids expresses exactly that:
// Prioritize only moves forward, and ids without a task are dropped, so when the
// current song has nothing left to download, the requested one takes the front
// by itself.
//
// The old code prioritized the requested id unconditionally, which is what let a
// late request for an already-finished song push the current song out of the
// only download slot.
func (s *Service) prioritizeForRequest(requestedId string) {
	head := s.currentSongId()
	if head == "" || head == requestedId {
		s.downloaderSvc.Prioritize(requestedId)
	} else {
		s.downloaderSvc.Prioritize(head, requestedId)
	}
}

// requestTemporary handles a video that is not in the playlist.
//
// Such a video is not ours to schedule: it comes from the room's random-play
// queue, so the software has no list to match it against, and it must not be
// preferred over the song that is playing. It gets a download task all the same
// — the queue is how a video gets cached at all — but that task starts at the
// tail and is never prioritized, and the request itself is answered by the
// origin instead of by an empty wait.
func (s *Service) requestTemporary(id string, ctx context.Context) (types.CDNResource, error) {
	item := song.GetTemporarySongByInternalId(id, ctx)

	if s.hasCompleteCache(id, ctx) {
		// Already downloadable from the cache: serve it, build nothing, queue
		// nothing, and do not touch the queue order. This is the only case in
		// which waiting for the cache entry is instant; a partially cached
		// entry can start serving too, and the hand-off below notices that it
		// is already initialized and cuts its own response at once.
		return s.getResource(item, ctx)
	}

	s.makeSureDownloading(item)

	return nil, ErrPassThrough
}

// currentSongId is the video the room is playing now.
//
// It is the head of the playlist, which is the same reading preload and
// healthCheck use. Nothing is returned when there is no playlist or it is
// empty; Prioritize drops ids without a task, so an empty id is simply ignored.
func (s *Service) currentSongId() string {
	items := s.pl.GetItemsSnapshot()
	if len(items) == 0 {
		return ""
	}

	return items[0].SongId()
}
