package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func CreatePreloadedPyPySong(id int) *PreloadedSong {
	song, ok := raw_song.FindPyPySong(id)
	if !ok {
		// maybe caused by corrupted song list
		cache.DownloadPyPySongs()
		logger.WarnLn("Cannot find PyPyDance song", id, "in the manifest")
		song = &raw_song.PyPyDanceSong{
			ID: id,
		}
	}
	idIncrement++
	ret := &PreloadedSong{
		sm:       NewSongStateMachine(),
		PyPySong: song,

		InfoNa: !ok,

		ID: idIncrement,

		em:     utils.NewEventManager[ChangeType](),
		lazyEm: utils.NewEventManager[ChangeType](),
	}
	ret.completeDuration()
	ret.sm.ps = ret
	return ret
}

func CreatePreloadedWannaSong(id int) *PreloadedSong {
	song, ok := raw_song.FindWannaSong(id)
	if !ok {
		// maybe caused by corrupted song list
		cache.DownloadWannaSongs()
		logger.WarnLn("Cannot find WannaDance song", id, "in the manifest")
		song = &raw_song.WannaDanceSong{
			DanceId: id,
		}
	}
	idIncrement++
	ret := &PreloadedSong{
		sm:        NewSongStateMachine(),
		WannaSong: song,

		InfoNa: !ok,

		ID: idIncrement,

		em:     utils.NewEventManager[ChangeType](),
		lazyEm: utils.NewEventManager[ChangeType](),
	}
	ret.completeDuration()
	ret.sm.ps = ret
	return ret
}

func CreatePreloadedCustomSong(url string) *PreloadedSong {
	idIncrement++
	ret := &PreloadedSong{
		sm:         NewSongStateMachine(),
		CustomSong: raw_song.FindOrCreateCustomSong(url),

		ID: idIncrement,

		em:     utils.NewEventManager[ChangeType](),
		lazyEm: utils.NewEventManager[ChangeType](),
	}
	ret.completeDuration()
	ret.completeTitle()
	ret.sm.ps = ret
	return ret
}

func CreateUnknownSong() *PreloadedSong {
	idIncrement++
	ret := &PreloadedSong{
		Unknown: true,
		ID:      idIncrement,

		sm: NewSongStateMachine(),

		em:     utils.NewEventManager[ChangeType](),
		lazyEm: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret
	ret.sm.DownloadStatus = NotAvailable
	return ret
}
