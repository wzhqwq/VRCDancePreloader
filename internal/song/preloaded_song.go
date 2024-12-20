package song

import (
	"fmt"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

type PreloadedSong struct {
	sm         *StateMachine
	PyPySong   *raw_song.PyPyDanceSong
	CustomSong *raw_song.CustomSong

	// constant
	Adder      string
	RandomPlay bool

	// play progress states
	Duration   float64
	TimePassed float64

	// download progress states
	TotalSize      int64
	DownloadedSize int64

	// diagnostic states
	PreloadError error

	// event
	em *EventManager
}

// constructors

func CreatePreloadedPyPySong(id int) *PreloadedSong {
	song, ok := raw_song.FindPyPySong(id)
	if !ok {
		return nil
	}
	ret := &PreloadedSong{
		sm: NewSongStateMachine(),

		Duration: float64(song.End),
		PyPySong: song,

		em: NewEventManager(),
	}
	ret.sm.ps = ret
	return ret
}

func CreatePreloadedCustomSong(title, url string) *PreloadedSong {
	ret := &PreloadedSong{
		sm: NewSongStateMachine(),

		CustomSong: raw_song.FindOrCreateCustomSong(title, url),

		em: NewEventManager(),
	}
	ret.sm.ps = ret
	return ret
}

func CreateRandomPlaySong() *PreloadedSong {
	ret := &PreloadedSong{
		RandomPlay: true,

		sm: NewSongStateMachine(),
		em: NewEventManager(),
	}
	ret.sm.ps = ret
	return ret
}

func CreateEmptySong() *PreloadedSong {
	ret := &PreloadedSong{
		RandomPlay: false,

		sm: NewSongStateMachine(),
		em: NewEventManager(),
	}
	ret.sm.ps = ret
	return ret
}

// getters

func (ps *PreloadedSong) GetOriginalUrl() string {
	if ps.PyPySong != nil && len(ps.PyPySong.OriginalURL) > 0 {
		return ps.PyPySong.OriginalURL[0]
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.Url
	}
	return ""
}
func (ps *PreloadedSong) GetDownloadUrl() string {
	if ps.PyPySong != nil {
		return utils.GetPyPyVideoUrl(ps.PyPySong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.Url
	}
	return ""
}
func (ps *PreloadedSong) GetThumbnailUrl() string {
	if ps.PyPySong != nil {
		return utils.GetPyPyThumbnailUrl(ps.PyPySong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.ThumbnailUrl
	}
	return ""
}
func (ps *PreloadedSong) GetId() string {
	if ps.PyPySong != nil {
		return fmt.Sprintf("pypy_%d", ps.PyPySong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.UniqueId
	}
	if ps.RandomPlay {
		return "random_play"
	}
	return "empty_song"
}
func (ps *PreloadedSong) GetPreloadStatus() DownloadStatus {
	return ps.sm.DownloadStatus
}

// compare

func (ps *PreloadedSong) MatchWithQueueItem(queueItem *types.QueueItem) bool {
	if queueItem.SongNum < 0 {
		return ps.CustomSong != nil && ps.CustomSong.MatchUrl(queueItem.URL)
	}
	if queueItem.SongNum == 0 {
		return ps.PyPySong == nil && ps.CustomSong == nil && !ps.RandomPlay
	}
	return ps.PyPySong != nil && ps.PyPySong.ID == queueItem.SongNum
}
func (ps *PreloadedSong) MatchWithCustomUrl(url string) bool {
	if ps.CustomSong == nil {
		return false
	}
	return ps.CustomSong.MatchUrl(url)
}
func (ps *PreloadedSong) MatchWithPyPyId(id int) bool {
	if ps.PyPySong == nil {
		return false
	}
	return ps.PyPySong.ID == id
}

// actions

func (ps *PreloadedSong) PlaySongStartFrom(offset float64) {
	ps.sm.PlaySongStartFrom(offset)
}
func (ps *PreloadedSong) PreloadSong() {
	if ps.GetDownloadUrl() == "" {
		return
	}
	if ps.sm.DownloadStatus == Initial || ps.sm.DownloadStatus == Failed {
		ps.sm.StartDownload()
	}
}
func (ps *PreloadedSong) PrioritizeSong() {
	if ps.sm.DownloadStatus == Pending {
		ps.sm.Prioritize()
	}
}
func (ps *PreloadedSong) RemoveFromList() {
	ps.sm.RemoveFromList()
}
