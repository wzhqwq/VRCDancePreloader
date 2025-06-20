package song

import (
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"time"
)

type PreloadedSong struct {
	sm         *StateMachine
	PyPySong   *raw_song.PyPyDanceSong
	CustomSong *raw_song.CustomSong
	WannaSong  *raw_song.WannaDanceSong

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
	em *utils.EventManager[ChangeType]
}

// constructors

func CreatePreloadedPyPySong(id int) *PreloadedSong {
	song, ok := raw_song.FindPyPySong(id)
	if !ok {
		// maybe caused by corrupted song list
		// TODO: reload song list to fix it
		log.Println("Warning: failed to find PyPyDance song ", id)
		song = &raw_song.PyPyDanceSong{
			ID:          id,
			Group:       0,
			Name:        fmt.Sprintf("PyPyDance #%d", id),
			Start:       0,
			End:         1,
			OriginalURL: []string{},
		}
	}
	ret := &PreloadedSong{
		sm: NewSongStateMachine(),

		Duration: float64(song.End),
		PyPySong: song,

		em: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret
	return ret
}

func CreatePreloadedWannaSong(id int) *PreloadedSong {
	song, ok := raw_song.FindWannaSong(id)
	if !ok {
		// maybe caused by corrupted song list
		// TODO: reload song list to fix it
		log.Println("Warning: failed to find WannaDance song ", id)
		song = &raw_song.WannaDanceSong{
			ID:    id,
			Group: "",
			Name:  fmt.Sprintf("WannaDance #%d", id),
			Start: 0,
			End:   1,
		}
	}
	ret := &PreloadedSong{
		sm: NewSongStateMachine(),

		Duration:  float64(song.End),
		WannaSong: song,

		em: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret
	return ret
}

func CreatePreloadedCustomSong(title, url string) *PreloadedSong {
	ret := &PreloadedSong{
		sm: NewSongStateMachine(),

		CustomSong: raw_song.FindOrCreateCustomSong(title, url),

		em: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret
	return ret
}

func CreateRandomPlaySong() *PreloadedSong {
	ret := &PreloadedSong{
		RandomPlay: true,

		sm: NewSongStateMachine(),
		em: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret
	return ret
}

func CreateUnknownSong() *PreloadedSong {
	ret := &PreloadedSong{
		RandomPlay: false,

		sm: NewSongStateMachine(),
		em: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret
	ret.sm.DownloadStatus = NotAvailable
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
	if ps.WannaSong != nil {
		return utils.GetWannaVideoUrl(ps.WannaSong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.Url
	}
	return ""
}
func (ps *PreloadedSong) GetId() string {
	if ps.PyPySong != nil {
		return fmt.Sprintf("pypy_%d", ps.PyPySong.ID)
	}
	if ps.WannaSong != nil {
		return fmt.Sprintf("wanna_%d", ps.WannaSong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.UniqueId
	}
	if ps.RandomPlay {
		return "random_play"
	}
	return "unknown"
}
func (ps *PreloadedSong) GetPreloadStatus() DownloadStatus {
	return ps.sm.DownloadStatus
}

// compare

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
func (ps *PreloadedSong) MatchWithWannaId(id int) bool {
	if ps.WannaSong == nil {
		return false
	}
	return ps.WannaSong.DanceId == id
}

// actions

func (ps *PreloadedSong) PlaySongStartFrom(offset float64) {
	ps.sm.PlaySongStartFrom(offset)
}
func (ps *PreloadedSong) PreloadSong() {
	if ps.sm.DownloadStatus == NotAvailable {
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
func (ps *PreloadedSong) AddToHistory() {
	info := ps.GetInfo()
	startTime := time.Now().Unix() - int64(ps.TimePassed)
	persistence.AddToHistory(info.ID, info.Title, ps.Adder, time.Unix(startTime, 0))
}
