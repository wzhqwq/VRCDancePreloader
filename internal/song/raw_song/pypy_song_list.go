package raw_song

import (
	"sync"
	"time"
)

var pypySongMap map[int]PyPyDanceSong
var pypyListMu sync.RWMutex
var pypyGroups []string
var pypyUpdateTime time.Time

func FindPyPySong(id int) (*PyPyDanceSong, bool) {
	pypyListMu.RLock()
	defer pypyListMu.RUnlock()

	if pypySongMap == nil {
		return nil, false
	}
	song, ok := pypySongMap[id]
	return &song, ok
}

func GetPyPyUpdateTime() (time.Time, bool) {
	if pypySongMap == nil {
		return time.Time{}, false
	}
	return pypyUpdateTime, true
}

type PyPyDanceListResponse struct {
	Timestamp int             `json:"timestamp"`
	Groups    []string        `json:"groups"`
	Songs     []PyPyDanceSong `json:"songs"`
}

func ProcessPyPyDanceList(data *PyPyDanceListResponse) {
	pypyListMu.Lock()
	defer pypyListMu.Unlock()

	if data == nil {
		return
	}

	pypySongMap = make(map[int]PyPyDanceSong)
	for _, song := range data.Songs {
		pypySongMap[song.ID] = song
	}
	pypyGroups = data.Groups
	pypyUpdateTime = time.Unix(int64(data.Timestamp/1000), 0)

	em.NotifySubscribers("PyPyDance")
}
