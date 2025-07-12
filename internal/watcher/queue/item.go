package queue

import (
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"regexp"
	"strings"
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
	//Length     int    `json:"length"`
	URL        string `json:"url"`
	PlayerName string `json:"playerName"`
	//Group      string `json:"group"`
}

func (item *PyPyQueueItem) ToPreloaded() *song.PreloadedSong {
	if item.SongNum > 0 {
		// PyPyDance Song
		return song.CreatePreloadedPyPySong(item.SongNum)
	}
	if item.SongNum < 0 && item.URL != "" {
		return song.CreatePreloadedCustomSong(item.URL)
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
	//Duration    int      `json:"duration"`
	//Group       string   `json:"group"`
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
		return song.CreatePreloadedWannaSong(item.SongID)
	}
	if item.SongID < 0 {
		url := extractUrlFromTitle(item.Title)
		if url != "" {
			return song.CreatePreloadedCustomSong(url)
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
