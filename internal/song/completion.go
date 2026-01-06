package song

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
)

func (ps *PreloadedSong) completeDuration() {
	if ps.Duration > 1 {
		return
	}
	if ps.PyPySong != nil {
		ps.Duration = time.Duration(ps.PyPySong.End) * time.Second
		return
	}
	if ps.WannaSong != nil {
		ps.Duration = time.Duration(ps.WannaSong.End) * time.Second
		return
	}
	if ps.DuDuSong != nil {
		ps.Duration = time.Duration(ps.DuDuSong.End) * time.Second
		return
	}
	if ps.CustomSong != nil {
		go func() {
			ps.Duration = third_party_api.GetDurationByInternalID(ps.CustomSong.UniqueId).Get()
		}()
	}
}

func (ps *PreloadedSong) completeTitle() {
	if ps.CustomSong != nil {
		go func() {
			id := ps.GetSongId()
			title := ps.CustomSong.Name

			// Try complete title using local database
			entry, err := persistence.GetEntry(id)
			if err == nil && entry.Title != title {
				ps.CustomSong.Name = entry.Title
				ps.notifyInfoChange()
			}

			// Try complete title using third party api
			completedTitle := third_party_api.CompleteTitleByInternalID(id, title).Get()
			if completedTitle != "" && completedTitle != title {
				ps.CustomSong.Name = completedTitle
				persistence.UpdateSavedTitle(id, completedTitle)
				ps.notifyInfoChange()
			}
		}()
	}
}

func (ps *PreloadedSong) UpdateSong() bool {
	var completedTitle string

	if ps.PyPySong != nil {
		song, ok := raw_song.FindPyPySong(ps.PyPySong.ID)
		if ok {
			ps.PyPySong = song
			completedTitle = song.Name
			goto complete
		}
	}
	if ps.WannaSong != nil {
		song, ok := raw_song.FindWannaSong(ps.WannaSong.DanceId)
		if ok {
			ps.WannaSong = song
			completedTitle = song.Name
			goto complete
		}
	}
	if ps.DuDuSong != nil {
		song, ok := raw_song.FindDuDuSong(ps.DuDuSong.ID)
		if ok {
			ps.DuDuSong = song
			completedTitle = song.Name
			goto complete
		}
	}
	if ps.CustomSong != nil {
		ps.completeDuration()
		ps.completeTitle()

		return true
	}
	return false

complete:
	ps.InfoNa = false
	persistence.UpdateSavedTitle(ps.GetSongId(), completedTitle)
	ps.completeDuration()
	ps.notifyInfoChange()

	return true
}
