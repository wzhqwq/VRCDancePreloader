package playlist

import (
	"errors"
	"io"
	"log"
	"slices"
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
)

var currentPlaylist []*PlayItem
var mutatingMutex = &sync.Mutex{}

var maxPreload int
var maxParallelDownload int

func Init(maxPreload_, maxParallelDownload_ int) {
	err := loadSongs()
	if err != nil {
		log.Println("Failed to load songs:", err)
		panic(err)
	}

	maxPreload = maxPreload_
	maxParallelDownload = maxParallelDownload_

	go keepCriticalUpdate()
}

func Stop() {
	currentPlaylist = nil
}

func GetSongsResponse() []byte {
	if savedResponse == nil {
		<-songLoaded
	}
	return savedResponse
}

func RestartPlaylist(items []*PlayItem) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()

	currentPlaylist = items
	for _, item := range currentPlaylist {
		gui.AddPlayItem(item)
	}
	CriticalUpdate()
	log.Println("Restarted playlist")
}

func RemoveItem(id int) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()

	removed := false
	for i, item := range currentPlaylist {
		if item.ID == id {
			currentPlaylist = slices.Delete(currentPlaylist, i, i+1)
			log.Println("Removed item", item.Title)
			removed = true
			gui.RemovePlayItem(item)
		}
		if removed {
			item.UpdateIndex(item.Index - 1)
		}
	}

	if removed {
		CriticalUpdate()
	}
}

func InsertItem(item *PlayItem, beforeId int) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()

	if beforeId == -1 {
		item.Index = len(currentPlaylist)
		currentPlaylist = append(currentPlaylist, item)
		log.Println("Appended item", item.Title)

		gui.AddPlayItem(item)
		CriticalUpdate()
	}
	beforeIndex := -1
	for i, item := range currentPlaylist {
		if item.ID == beforeId {
			beforeIndex = i
			item.Index = i
		}
		if beforeIndex != -1 {
			item.UpdateIndex(item.Index + 1)
		}
	}

	if beforeIndex != -1 {
		currentPlaylist = slices.Insert(currentPlaylist, beforeIndex, item)
		log.Println("Inserted item", item.Title, "before", beforeId)

		gui.AddPlayItem(item)
		CriticalUpdate()
	}
}

func createFromSongList(id int) *PlayItem {
	if song, ok := songMap[id]; ok {
		return NewPlayItem(song.Name, songGroups[song.Group], i18n.T("placeholder_unknown_adder"), "", song.ID, song.End, -1)
	}
	return nil
}

func PlaySong(id int) (io.ReadSeekCloser, error) {
	mutatingMutex.Lock()
	for index, item := range currentPlaylist {
		if item.ID == id {
			reader, err := item.ToReader()
			if err != nil {
				return nil, err
			}
			item.UpdateStatus(constants.Playing)
			for i := 0; i < index; i++ {
				if currentPlaylist[i].Status == constants.Playing {
					currentPlaylist[i].UpdateStatus(constants.Ended)
				}
			}
			return reader, nil
		}
	}
	mutatingMutex.Unlock()

	// it won't be rendered, but it will be downloaded
	if tempItem := createFromSongList(id); tempItem != nil {
		reader, err := tempItem.ToReader()
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	return nil, errors.New("Song not found")
}
