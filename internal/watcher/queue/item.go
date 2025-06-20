package queue

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"regexp"
	"strings"
)

type QueueItem interface {
	ToPreloaded() *song.PreloadedSong
	MatchWithPreloaded(song *song.PreloadedSong) bool
	GetAdder() string
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
	if item.SongNum < 0 {
		// Custom Song
		return song.CreatePreloadedCustomSong(item.VideoName, item.URL)
	}
	return song.CreateUnknownSong()
}

func (item *PyPyQueueItem) MatchWithPreloaded(song *song.PreloadedSong) bool {
	if item.SongNum < 0 {
		return song.CustomSong != nil && song.CustomSong.MatchUrl(item.URL)
	}
	if item.SongNum == 0 {
		return song.PyPySong == nil && song.CustomSong == nil && !song.RandomPlay
	}
	return song.PyPySong != nil && song.PyPySong.ID == item.SongNum
}

func (item *PyPyQueueItem) GetAdder() string {
	return item.PlayerName
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

func (item *WannaQueueItem) ToPreloaded() *song.PreloadedSong {
	if item.SongID > 0 {
		// WannaDance Song
		return song.CreatePreloadedWannaSong(item.SongID)
	}
	if item.SongID < 0 {
		matches := regexp.MustCompile(`URL: (.*)`).FindSubmatch([]byte(item.Title))
		if len(matches) > 0 {
			url := string(matches[1])
			if url[:2] == "BV" {
				// BiliBili
				return song.CreatePreloadedCustomSong(url, utils.GetStandardBiliURL(url))
			}
		}
	}
	return song.CreateUnknownSong()
}

func (item *WannaQueueItem) MatchWithPreloaded(song *song.PreloadedSong) bool {
	return song.WannaSong != nil && song.WannaSong.DanceId == item.SongID
}

func (item *WannaQueueItem) GetAdder() string {
	if item.Random {
		return "Random"
	}
	return strings.Join(item.PlayerNames, ",")
}
