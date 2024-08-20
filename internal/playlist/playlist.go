package playlist

import (
	"io"
	"log"
	"os"
	"slices"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
)

var currentPlaylist []*PlayItem

func Init() {
	err := loadSongs()
	if err != nil {
		log.Println("Failed to load songs:", err)
		panic(err)
	}
	go pw.Render()
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

func CriticalUpdate() {
	// for _, item := range currentPlaylist {
	// 	log.Println(item.ID, item.Title, item.Status)
	// }
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Title", "Status"})
	t.AppendRows(lo.Map(currentPlaylist, func(item *PlayItem, _ int) table.Row {
		if item.ID >= 0 {
			return table.Row{item.ID, item.Title, item.Status}
		}
		return table.Row{"Custom", item.Title, item.Status}
	}))
	t.Render()
	PreloadPlaylist()
}

func PreloadPlaylist() {
	scanned := 0
	for _, item := range currentPlaylist {
		if scanned >= 4 {
			break
		}
		if item.Status == Ended || item.Status == Playing {
			continue
		}
		if item.Status == Failed {
			item.UpdateStatus(Pending)
		}
		if item.Status == Pending && item.ID >= 0 {
			go func() {
				defer CriticalUpdate()

				item.Download()
				item.WaitForPlay()
			}()
		}
		scanned++
	}
}

func RestartPlaylist(items []*PlayItem) {
	defer CriticalUpdate()

	currentPlaylist = items
	log.Println("Restarted playlist")
}

func RemoveItem(id int) {
	defer CriticalUpdate()

	currentPlaylist = lo.Filter(currentPlaylist, func(item *PlayItem, _ int) bool {
		return item.ID != id
	})
	log.Println("Removed item", id)
}

func InsertItem(item *PlayItem, beforeId int) {
	defer CriticalUpdate()

	if beforeId == -1 {
		currentPlaylist = append(currentPlaylist, item)
		log.Println("Appended item", item.Title)
		return
	}
	for i, v := range currentPlaylist {
		if v.ID == beforeId {
			currentPlaylist = slices.Insert(currentPlaylist, i, item)
			log.Println("Inserted item", item.Title, "before", beforeId)
			return
		}
	}
}

func createFromSongList(id int) *PlayItem {
	if song, ok := songMap[id]; ok {
		return NewPlayItem(song.Name, songGroups[song.Group], "", "", song.ID, song.End)
	}
	return nil
}

func PlaySong(id int) (int, io.ReadSeekCloser) {
	for index, item := range currentPlaylist {
		if item.ID == id {
			reader, err := item.ToReader()
			if err != nil {
				log.Println("Failed to copy song from cache:", err)
				return 0, nil
			}
			item.UpdateStatus(Playing)
			for i := 0; i < index; i++ {
				if currentPlaylist[i].Status == Playing {
					currentPlaylist[i].UpdateStatus(Ended)
				}
			}
			return item.Size, reader
		}
	}

	// create temperate
	if item := createFromSongList(id); item != nil {
		reader, err := item.ToReader()
		if err != nil {
			log.Println("Failed to copy song from cache:", err)
			return 0, nil
		}
		return item.Size, reader
	}

	return 0, nil
}
