package raw_song

import (
	"sync"
	"time"
)

var wannaSongMap map[int]WannaDanceSong
var wannaListMu sync.RWMutex
var wannaUpdateTime time.Time

func FindWannaSong(id int) (*WannaDanceSong, bool) {
	wannaListMu.RLock()
	defer wannaListMu.RUnlock()

	if wannaSongMap == nil {
		return nil, false
	}
	song, ok := wannaSongMap[id]
	return &song, ok
}

func GetWannaUpdateTime() (time.Time, bool) {
	if wannaSongMap == nil {
		return time.Time{}, false
	}
	return wannaUpdateTime, true
}

type WannaDanceListResponse struct {
	Time   string           `json:"time"`
	Groups WannaDanceGroups `json:"groups"`
}
type WannaDanceGroups struct {
	Contents []WannaDanceGroupContent `json:"contents"`
}
type WannaDanceGroupContent struct {
	Name  string           `json:"groupName"`
	Major string           `json:"major"`
	Songs []WannaDanceSong `json:"songInfos"`
}

func ProcessWannaDanceList(data *WannaDanceListResponse) {
	wannaListMu.Lock()
	defer wannaListMu.Unlock()

	if data == nil {
		return
	}

	wannaSongMap = make(map[int]WannaDanceSong)

	for _, group := range data.Groups.Contents {
		name := group.Name
		for _, song := range group.Songs {
			song.Group = name
			wannaSongMap[song.DanceId] = song
		}
	}
	wannaUpdateTime, _ = time.Parse("20060102150405", data.Time)

	em.NotifySubscribers("WannaDance")
}
