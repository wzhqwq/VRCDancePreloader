package queue

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type QueueItem interface {
	ToPreloaded() *song.PreloadedSong
	MatchWithPreloaded(song *song.PreloadedSong) bool
	GetAdder() string
	ToString() string
}

type PyPyQueueItem struct {
	QueueItem

	SongNum   int    `json:"songNum"`
	VideoName string `json:"videoName"`
	Length    int    `json:"length"`
	URL       string `json:"url"`
	// ThumbnailUrl string `json:"thumbnailUrl"`
	PlayerName string `json:"playerName"`
	// PlayerColor string `json:"playerColor"`
	Group string `json:"group"`
}

func (item *PyPyQueueItem) ToPreloaded() *song.PreloadedSong {
	if item.SongNum > 0 {
		// PyPyDance Song
		s := song.GetPyPySongForList(item.SongNum)
		// Try to complete the info with the queue item
		if s.InfoNa && item.VideoName != "" && item.Group != "" && item.Length > 0 {
			s.PyPySong.Complete(item.VideoName, item.Group, item.Length)
			s.InfoNa = false
		}
		return s
	}
	if item.SongNum < 0 && item.URL != "" {
		return song.GetCustomSongForList(item.URL)
	}
	return song.CreateUnknownSong()
}

func (item *PyPyQueueItem) MatchWithPreloaded(song *song.PreloadedSong) bool {
	if item.SongNum > 0 {
		return song.MatchWithPyPyId(item.SongNum)
	}
	if item.SongNum < 0 && item.URL != "" {
		return song.MatchWithCustomUrl(item.URL)
	}
	return song.Unknown
}

func (item *PyPyQueueItem) GetAdder() string {
	return item.PlayerName
}

func (item *PyPyQueueItem) ToString() string {
	return fmt.Sprintf("pypy_%d", item.SongNum)
}

type WannaQueueItem struct {
	QueueItem

	PlayerNames []string `json:"playerNames"`
	Title       string   `json:"title"`
	//PlayerCount string   `json:"playerCount"`
	SongID int `json:"songId"`
	//Major       string   `json:"major"`
	Duration int    `json:"duration"`
	Group    string `json:"group"`
	//DoubleWidth bool     `json:"doubleWidth"`

	Random bool
}

func extractUrlFromTitle(title string) string {
	matches := regexp.MustCompile(`URL: (.+)`).FindSubmatch([]byte(title))
	if len(matches) > 0 {
		url := string(matches[1])
		if len(url) > 2 && url[:2] == "BV" {
			// BiliBili
			return utils.GetStandardBiliURL(url)
		}
	}
	return ""
}

func (item *WannaQueueItem) ToPreloaded() *song.PreloadedSong {
	if item.SongID > 0 {
		s := song.GetWannaSongForList(item.SongID)
		// Try to complete the info with the queue item
		if s.InfoNa && item.Title != "" && item.Group != "" && item.Duration > 0 {
			prefix := fmt.Sprintf("%d. ", item.SongID)
			item.Title = strings.TrimPrefix(item.Title, prefix)
			s.WannaSong.Complete(item.Title, item.Group, item.Duration)
			s.InfoNa = false
		}
		return s
	}
	if item.SongID < 0 {
		url := extractUrlFromTitle(item.Title)
		if url != "" {
			return song.GetCustomSongForList(url)
		}
	}
	return song.CreateUnknownSong()
}

func (item *WannaQueueItem) MatchWithPreloaded(song *song.PreloadedSong) bool {
	if item.SongID > 0 {
		return song.MatchWithWannaId(item.SongID)
	}
	if item.SongID < 0 {
		url := extractUrlFromTitle(item.Title)
		if url != "" {
			return song.MatchWithCustomUrl(url)
		}
	}
	return song.Unknown
}

func (item *WannaQueueItem) GetAdder() string {
	if item.Random {
		return "Random"
	}
	return strings.Join(item.PlayerNames, ",")
}

func (item *WannaQueueItem) ToString() string {
	return fmt.Sprintf("wanna_%d", item.SongID)
}

type DuDuQueueItem struct {
	Title      string `json:"title"`
	PlayerName string `json:"playerName"`
	Group      string `json:"group"`
	GroupName  string `json:"groupName"`
	Duration   int    `json:"duration"`
	// SongVolume int    `json:"songVolume"`
	SongID int `json:"songId"`

	Random bool
}

func (item *DuDuQueueItem) ToPreloaded() *song.PreloadedSong {
	if item.SongID >= 0 {
		s := song.GetDuDuSongForList(item.SongID)
		// Try to complete the info with the queue item
		if s.InfoNa && item.Title != "" && item.Group != "" && item.Duration > 0 {
			prefix := fmt.Sprintf("%d. ", item.SongID)
			if strings.HasPrefix(item.Title, prefix) {
				item.Title = strings.TrimPrefix(item.Title, prefix)
			}
			s.DuDuSong.Complete(item.Title, item.Group, item.Duration)
			s.InfoNa = false
		}
		return s
	}
	if item.SongID < 0 {
		url := extractUrlFromTitle(item.Title)
		if url != "" {
			return song.GetCustomSongForList(url)
		}
	}
	return song.CreateUnknownSong()
}

func (item *DuDuQueueItem) MatchWithPreloaded(song *song.PreloadedSong) bool {
	if item.SongID >= 0 {
		return song.MatchWithDuDuId(item.SongID)
	}
	if item.SongID < 0 {
		url := extractUrlFromTitle(item.Title)
		if url != "" {
			return song.MatchWithCustomUrl(url)
		}
	}
	return song.Unknown
}

func (item *DuDuQueueItem) GetAdder() string {
	return item.PlayerName
}

func (item *DuDuQueueItem) ToString() string {
	return fmt.Sprintf("dudu_%d", item.SongID)
}
