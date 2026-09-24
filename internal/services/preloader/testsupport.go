package preloader

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/playlist"
)

// Test support.
//
// The decision this package makes about the download queue — "the current song
// first, the requested one after it, and never the other way around" — is what
// the preloader does for every video the room asks for, so it has to be asserted
// here rather than in the downloader. Serving the request itself needs the whole
// application (a resolved remote, a cache directory, the providers' catalog),
// which is not what that assertion is about, so the two are separated below.
//
// These are not part of the service API and must not be used by production code.

// UsePlaylistForTest installs the playlist the service answers requests from.
func (s *Service) UsePlaylistForTest(pl *playlist.PlayList) {
	s.plMu.Lock()
	defer s.plMu.Unlock()

	s.pl = pl
}

// RequestQueueOrderForTest runs the queue decision Request makes for a video
// that is in the playlist, and returns the resulting order.
//
// It is Request without serveResource: the same lookup, the same
// makeSureDownloading and the same priority rule, so a change to the rule shows
// up here. It reports false when the id is not in the playlist, which is the
// branch that goes to the origin instead.
func (s *Service) RequestQueueOrderForTest(id string) ([]string, bool) {
	s.plMu.RLock()
	defer s.plMu.RUnlock()

	if s.pl == nil {
		return nil, false
	}

	item := s.pl.SearchByInternalId(id)
	if item == nil {
		return nil, false
	}

	s.makeSureDownloading(item)
	s.prioritizeForRequest(id)

	return s.downloaderSvc.QueueOrderForTest(), true
}
