package playlist

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func (pl *PlayList) SearchByUrl(url string) *song.PreloadedSong {
	if id, ok := utils.CheckPyPyUrl(url); ok {
		return pl.FindPyPySong(id)
	}
	if id, ok := utils.CheckWannaUrl(url); ok {
		return pl.FindWannaSong(id)
	}
	if id, ok := utils.CheckDuDuUrl(url); ok {
		return pl.FindDuDuSong(id)
	}
	return pl.FindCustomSong(url)
}
