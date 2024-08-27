package watcher

import (
	"encoding/json"

	"github.com/samber/lo"

	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

var currentQueue []types.QueueItem = make([]types.QueueItem, 0)

func parseQueue(data []byte) ([]types.QueueItem, error) {
	// parse the json data into a slice of QueueItem
	var items []types.QueueItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func diffQueues(old, new []types.QueueItem) {
	// do the lcs
	lengths := make([][]int, len(old)+1)
	for i := 0; i <= len(old); i++ {
		lengths[i] = make([]int, len(new)+1)
	}

	// row 0 and column 0 are initialized to 0 already
	for i := 0; i < len(old); i++ {
		for j := 0; j < len(new); j++ {
			match := old[i].SongNum == new[j].SongNum
			if old[i].SongNum == -1 || new[j].SongNum == -1 {
				match = old[i].URL == new[j].URL
			}
			if match {
				lengths[i+1][j+1] = lengths[i][j] + 1
			} else if lengths[i+1][j] > lengths[i][j+1] {
				lengths[i+1][j+1] = lengths[i+1][j]
			} else {
				lengths[i+1][j+1] = lengths[i][j+1]
			}
		}
	}

	if lengths[len(old)][len(new)] == 0 {
		// if the lcs is zero, then we consider it as a new queue
		// clear the current queue
		list := lo.Map(new, func(item types.QueueItem, _ int) *playlist.PlayItem {
			return playlist.CreateFromQueueItem(item)
		})
		playlist.RestartPlaylist(list)
		return
	}

	// read the substring out from the matrix
	x, y := len(old), len(new)
	for x > 0 || y > 0 {
		if x > 0 && lengths[x][y] == lengths[x-1][y] {
			x--
			playlist.RemoveItem(&old[x])
		} else if y > 0 && lengths[x][y] == lengths[x][y-1] {
			y--
			newItem := playlist.CreateFromQueueItem(new[y])
			if x == len(old) {
				playlist.InsertItem(newItem, nil)
			} else {
				playlist.InsertItem(newItem, &old[x])
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
