package playlist

import (
	"errors"
	"io"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

var currentPlaylist []*PlayItem
var temporaryItem *PlayItem
var mutatingMutex = &sync.Mutex{}

var maxPreload int

func Init(maxPreload_ int) {
	err := loadSongs()
	if err != nil {
		log.Println("Failed to load songs:", err)
		panic(err)
	}

	maxPreload = maxPreload_
	currentPlaylist = make([]*PlayItem, 0)

	go keepCriticalUpdate()
}

func Stop() {
	for _, item := range currentPlaylist {
		item.Dispose()
	}
	currentPlaylist = nil
	<-time.After(1 * time.Second)
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
	if currentPlaylist == nil {
		return
	}

	for _, item := range currentPlaylist {
		item.Dispose()
		gui.RemovePlayItem(item)
	}
	currentPlaylist = items
	for index, item := range currentPlaylist {
		item.Index = index
		gui.AddPlayItem(item)
	}
	CriticalUpdate()
	log.Println("Restarted playlist")
}

func matchWithQueueItem(item *PlayItem, queueItem *types.QueueItem) bool {
	if queueItem.SongNum == -1 {
		return item.URL == queueItem.URL
	}
	return item.ID == queueItem.SongNum
}

func RemoveItem(queueData *types.QueueItem) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()
	if currentPlaylist == nil {
		return
	}

	removedIndex := -1
	for i, item := range currentPlaylist {
		if matchWithQueueItem(item, queueData) {
			removedIndex = i
			currentPlaylist = slices.Delete(currentPlaylist, i, i+1)
			item.Dispose()
			gui.RemovePlayItem(item)
			log.Println("Removed item", item.Title)
			break
		}
	}
	if removedIndex != -1 {
		for i := removedIndex; i < len(currentPlaylist); i++ {
			currentPlaylist[i].UpdateIndex(i)
		}
		CriticalUpdate()
	}
}

func InsertItem(item *PlayItem, before *types.QueueItem) {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()
	if currentPlaylist == nil {
		return
	}

	if before == nil {
		item.Index = len(currentPlaylist)
		currentPlaylist = append(currentPlaylist, item)
		gui.AddPlayItem(item)
		log.Println("Appended item", item.Title)

		CriticalUpdate()
		return
	}
	insertIndex := -1
	for i, item := range currentPlaylist {
		if matchWithQueueItem(item, before) {
			insertIndex = i
			currentPlaylist = slices.Insert(currentPlaylist, insertIndex, item)
			item.Index = i
			gui.AddPlayItem(item)
			log.Println("Inserted item", item.Title, "before", before.VideoName)
			break
		}
	}
	if insertIndex != -1 {
		for i := insertIndex; i < len(currentPlaylist); i++ {
			currentPlaylist[i].UpdateIndex(i + 1)
		}
		CriticalUpdate()
	}
}

func createFromSongList(id int) *PlayItem {
	if song, ok := songMap[id]; ok {
		return NewPlayItem(
			song.Name,
			songGroups[song.Group],
			i18n.T("placeholder_unknown_adder"),
			"",
			song.ID,
			song.End,
			-1,
		)
	}
	return nil
}
func CreateFromQueueItem(item types.QueueItem) *PlayItem {
	if temporaryItem != nil && temporaryItem.ID == item.SongNum {
		temporaryItem.Adder = item.PlayerName
		temporaryItem.URL = item.URL
		item := temporaryItem
		temporaryItem = nil
		return item
	}

	return NewPlayItem(item.VideoName,
		item.Group,
		item.PlayerName,
		item.URL,
		item.SongNum,
		item.Length,
		-1,
	)
}

func FindSongToPlay(id int) *PlayItem {
	mutatingMutex.Lock()
	defer mutatingMutex.Unlock()

	for index, item := range currentPlaylist {
		if item.ID == id {
			item.UpdateStatus(constants.Playing)
			for i := 0; i < index; i++ {
				if currentPlaylist[i].Status == constants.Playing {
					currentPlaylist[i].UpdateStatus(constants.Ended)
				}
			}
			return item
		}
	}
	if item := createFromSongList(id); item != nil {
		if temporaryItem != nil {
			temporaryItem.Dispose()
		}
		temporaryItem = item
		return item
	}
	return nil
}

func PlaySong(id int) (io.ReadSeekCloser, error) {
	item := FindSongToPlay(id)
	if item == nil {
		return nil, errors.New("Song not found")
	}

	reader, err := item.ToReader()
	if err != nil {
		return nil, err
	}
	return reader, nil
}
