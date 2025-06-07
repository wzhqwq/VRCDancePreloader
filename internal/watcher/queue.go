package watcher

import (
	"encoding/json"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

func parseQueue(data []byte) ([]types.QueueItem, error) {
	// parse the json data into a slice of QueueItem
	var items []types.QueueItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func diffQueues(old []*song.PreloadedSong, new []types.QueueItem) {
	// do the lcs
	lengths := make([][]int, len(old)+1)
	for i := 0; i <= len(old); i++ {
		lengths[i] = make([]int, len(new)+1)
	}

	// row 0 and column 0 are initialized to 0 already
	for i := 0; i < len(old); i++ {
		for j := 0; j < len(new); j++ {
			match := old[i].MatchWithQueueItem(&new[j])
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
		playlist.ClearAndSetQueue(new)
		return
	}

	// read the substring out from the matrix
	x, y := len(old), len(new)
	for x > 0 || y > 0 {
		if x > 0 && lengths[x][y] == lengths[x-1][y] {
			x--
			playlist.RemoveItem(x)
		} else if y > 0 && lengths[x][y] == lengths[x][y-1] {
			y--
			if x == len(old) {
				playlist.InsertItem(new[y], -1)
			} else if x == 0 {
				// inserting before currently playing song is prohibited
				// instead we should clear and refill the queue
				// to prevent an incorrect playing state
				playlist.ClearAndSetQueue(new)
				return
			} else {
				playlist.InsertItem(new[y], x)
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
	diffQueues(playlist.GetQueue(), newQueue)

	return nil
}
