package watcher

import (
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
	"log"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

func diffQueues(old []*song.PreloadedSong, new []queue.QueueItem) {
	log.Println(
		lo.Map(old, func(item *song.PreloadedSong, _ int) string {
			return item.GetId()
		}),
		"->",
		lo.Map(new, func(item queue.QueueItem, _ int) string {
			return item.ToString()
		}),
	)
	// do the lcs
	lengths := make([][]int, len(old)+1)
	for i := 0; i <= len(old); i++ {
		lengths[i] = make([]int, len(new)+1)
	}

	// row 0 and column 0 are initialized to 0 already
	for i := 0; i < len(old); i++ {
		for j := 0; j < len(new); j++ {
			match := new[j].MatchWithPreloaded(old[i])
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
			if x == 0 {
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
