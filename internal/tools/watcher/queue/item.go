package queue

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

type Item interface {
	ToPreloaded() *song.StatefulSong
	MatchWithPreloaded(song *song.StatefulSong) bool
	GetAdder() string
	ToString() string
}

// PyPyDance

type PyPyQueueItem struct {
	Item

	SongNum   int    `json:"songNum"`
	VideoName string `json:"videoName"`
	Length    int    `json:"length"`
	URL       string `json:"url"`
	// ThumbnailUrl string `json:"thumbnailUrl"`
	PlayerName string `json:"playerName"`
	// PlayerColor string `json:"playerColor"`
	Group string `json:"group"`
}

func (item *PyPyQueueItem) ToPreloaded() *song.StatefulSong {
	if item.SongNum > 0 {
		s := song.MakePlaylistSongByInternalId(item.ToString())
		s.CompleteInfoIfEmpty(item.VideoName, item.Group, item.Length)
		return s
	}
	if item.SongNum < 0 && item.URL != "" {
		return song.MakePlaylistSongByInternalId(raw_song.GetInternalIdByUrl(item.URL))
	}
	return song.CreateUnknownSong()
}

func (item *PyPyQueueItem) MatchWithPreloaded(song *song.StatefulSong) bool {
	if item.SongNum > 0 {
		return song.SongId() == item.ToString()
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
	return internal_id.PyPyInternalPrefix + strconv.Itoa(item.SongNum)
}

// WannaDance

type WannaQueueItem struct {
	Item

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
			return internal_id.GetStandardBiliURL(url)
		}
	}
	return ""
}

func (item *WannaQueueItem) ToPreloaded() *song.StatefulSong {
	if item.SongID > 0 {
		s := song.MakePlaylistSongByInternalId(item.ToString())
		item.Title = strings.TrimPrefix(item.Title, fmt.Sprintf("%d. ", item.SongID))
		s.CompleteInfoIfEmpty(item.Title, item.Group, item.Duration)
		return s
	}
	if item.SongID < 0 {
		url := extractUrlFromTitle(item.Title)
		if url != "" {
			return song.MakePlaylistSongByInternalId(raw_song.GetInternalIdByUrl(url))
		}
	}
	return song.CreateUnknownSong()
}

func (item *WannaQueueItem) MatchWithPreloaded(song *song.StatefulSong) bool {
	if item.SongID > 0 {
		return song.SongId() == item.ToString()
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
	return internal_id.WannaInternalPrefix + strconv.Itoa(item.SongID)
}

// DuDuFitDance

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

func (item *DuDuQueueItem) ToPreloaded() *song.StatefulSong {
	if item.SongID >= 0 {
		s := song.MakePlaylistSongByInternalId(item.ToString())
		item.Title = strings.TrimPrefix(item.Title, fmt.Sprintf("%d. ", item.SongID))
		s.CompleteInfoIfEmpty(item.Title, item.Group, item.Duration)
		return s
	}
	if item.SongID < 0 {
		url := extractUrlFromTitle(item.Title)
		if url != "" {
			return song.MakePlaylistSongByInternalId(raw_song.GetInternalIdByUrl(url))
		}
	}
	return song.CreateUnknownSong()
}

func (item *DuDuQueueItem) MatchWithPreloaded(song *song.StatefulSong) bool {
	if item.SongID >= 0 {
		return song.SongId() == item.ToString()
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
	return internal_id.DuDuInternalPrefix + strconv.Itoa(item.SongID)
}
