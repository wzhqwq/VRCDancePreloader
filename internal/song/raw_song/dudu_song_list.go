package raw_song

import (
	"sync"
	"time"
)

var duduSongMap map[int]DuDuFitDanceSong
var duduListMu sync.RWMutex
var duduUpdateTime time.Time

func FindDuDuSong(id int) (*DuDuFitDanceSong, bool) {
	duduListMu.RLock()
	defer duduListMu.RUnlock()

	if duduSongMap == nil {
		return nil, false
	}
	song, ok := duduSongMap[id]
	return &song, ok
}

func GetDuDuUpdateTime() (time.Time, bool) {
	if duduSongMap == nil {
		return time.Time{}, false
	}
	return duduUpdateTime, true
}

type DuDuFitDanceListResponse struct {
	Timestamp int64               `json:"timestamp"`
	Groups    []DuDuFitDanceGroup `json:"groups"`
	Videos    []DuDuFitDanceSong  `json:"videos"`
}
type DuDuFitDanceGroup struct {
	Name   string `json:"name"`
	Videos []int  `json:"videos"`
}

func ProcessDuDuFitDanceList(data *DuDuFitDanceListResponse) {
	duduListMu.Lock()
	defer duduListMu.Unlock()

	if data == nil {
		return
	}

	duduSongMap = make(map[int]DuDuFitDanceSong)
	for _, song := range data.Videos {
		duduSongMap[song.ID] = song
	}
	for _, group := range data.Groups {
		name := group.Name
		for _, id := range group.Videos {
			song, ok := duduSongMap[id]
			if ok {
				song.Group = name
			}
		}
	}
	duduUpdateTime = time.Unix(data.Timestamp, 0)

	em.NotifySubscribers("DuDuFitDance")
}
