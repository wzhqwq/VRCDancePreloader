package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var idIncrement int64 = 0

func constructBasicPreloadedSong() *PreloadedSong {
	idIncrement++
	ret := &PreloadedSong{
		sm: NewSongStateMachine(),

		ID: idIncrement,

		em:     utils.NewEventManager[ChangeType](),
		lazyEm: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret

	return ret
}

func CreatePreloadedPyPySong(id int) *PreloadedSong {
	ret := constructBasicPreloadedSong()

	song, ok := raw_song.FindPyPySong(id)
	if !ok {
		// maybe caused by corrupted song list
		cache.DownloadPyPySongs()
		songLogger.WarnLn("Cannot find PyPyDance song", id, "in the manifest")
		song = &raw_song.PyPyDanceSong{
			ID: id,
		}
		ret.InfoNa = true
	}

	ret.PyPySong = song
	ret.completeDuration()

	return ret
}

func CreatePreloadedWannaSong(id int) *PreloadedSong {
	ret := constructBasicPreloadedSong()

	song, ok := raw_song.FindWannaSong(id)
	if !ok {
		// maybe caused by corrupted song list
		cache.DownloadWannaSongs()
		songLogger.WarnLn("Cannot find WannaDance song", id, "in the manifest")
		song = &raw_song.WannaDanceSong{
			DanceId: id,
		}
		ret.InfoNa = true
	}

	ret.WannaSong = song
	ret.completeDuration()

	return ret
}

func CreatePreloadedDuDuSong(id int) *PreloadedSong {
	ret := constructBasicPreloadedSong()

	song, ok := raw_song.FindDuDuSong(id)
	if !ok {
		// maybe caused by corrupted song list
		cache.DownloadDuDuSongs()
		songLogger.WarnLn("Cannot find DuDuFitDance song", id, "in the manifest")
		song = &raw_song.DuDuFitDanceSong{
			ID: id,
		}
		ret.InfoNa = true
	}

	ret.DuDuSong = song
	ret.completeDuration()

	return ret
}

func CreatePreloadedCustomSong(url string) *PreloadedSong {
	ret := constructBasicPreloadedSong()

	ret.CustomSong = raw_song.FindOrCreateCustomSong(url)
	ret.completeDuration()
	ret.completeTitle()

	return ret
}

func CreateUnknownSong() *PreloadedSong {
	ret := constructBasicPreloadedSong()

	ret.InfoNa = true
	ret.Unknown = true
	ret.sm.DownloadStatus = NotAvailable

	return ret
}
