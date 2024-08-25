package watcher

import (
	"encoding/json"
	"log"

	"github.com/samber/lo"

	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
)

var currentQueue []QueueItem = make([]QueueItem, 0)

// define a json structure named QueueItem
// {"songNum": 363, "videoName": "Monster - Lady Gaga", "length": 261, "url": "", "playerName": "NekoYama_Shiro", "group": "Fitness Marshall"}
type QueueItem struct {
	SongNum    int    `json:"songNum"`
	VideoName  string `json:"videoName"`
	Length     int    `json:"length"`
	URL        string `json:"url"`
	PlayerName string `json:"playerName"`
	Group      string `json:"group"`
}

func parseQueue(data []byte) ([]QueueItem, error) {
	// parse the json data into a slice of QueueItem
	var items []QueueItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func diffQueues(old, new []QueueItem) {
	// do the lcs
	lengths := make([][]int, len(old)+1)
	for i := 0; i <= len(old); i++ {
		lengths[i] = make([]int, len(new)+1)
	}

	// row 0 and column 0 are initialized to 0 already
	for i := 0; i < len(old); i++ {
		for j := 0; j < len(new); j++ {
			if old[i].SongNum == new[j].SongNum {
				lengths[i+1][j+1] = lengths[i][j] + 1
			} else if lengths[i+1][j] > lengths[i][j+1] {
				lengths[i+1][j+1] = lengths[i+1][j]
			} else {
				lengths[i+1][j+1] = lengths[i][j+1]
			}
		}
	}

	log.Println("lcs:", lengths[len(old)][len(new)])

	if lengths[len(old)][len(new)] == 0 {
		// if the lcs is zero, then we consider it as a new queue
		// clear the current queue
		playlist.RestartPlaylist(lo.Map(new, func(item QueueItem, index int) *playlist.PlayItem {
			return playlist.NewPlayItem(
				item.VideoName,
				item.Group,
				item.PlayerName,
				item.URL,
				item.SongNum,
				item.Length,
				index,
			)
		}))
		return
	}

	// read the substring out from the matrix
	x, y := len(old), len(new)
	for x > 0 || y > 0 {
		if x > 0 && lengths[x][y] == lengths[x-1][y] {
			x--
			playlist.RemoveItem(old[x].SongNum)
		} else if y > 0 && lengths[x][y] == lengths[x][y-1] {
			y--
			newItem := playlist.NewPlayItem(new[y].VideoName,
				new[y].Group,
				new[y].PlayerName,
				new[y].URL,
				new[y].SongNum,
				new[y].Length,
				-1,
			)
			if x == len(old) {
				playlist.InsertItem(newItem, -1)
			} else {
				playlist.InsertItem(newItem, old[x].SongNum)
			}
		} else if x > 0 && y > 0 {
			x--
			y--
		}
	}
}

func processQueueLog(data []byte) error {
	// parse the new queue
	newQueue, err := parseQueue(data)
	if err != nil {
		return err
	}

	// compare the new queue with the current queue
	diffQueues(currentQueue, newQueue)

	// update the current queue
	currentQueue = newQueue

	return nil
}
