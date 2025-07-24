package watcher

import (
	"context"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
	"log"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type queueMutation struct {
	index int
	item  queue.QueueItem
}

func insertMutation(index int, item queue.QueueItem) queueMutation {
	return queueMutation{index: index, item: item}
}
func deleteMutation(index int) queueMutation {
	return queueMutation{index: index}
}

func diffQueues(old []*song.PreloadedSong, new []queue.QueueItem) {
	if len(old) == len(new) {
		allTheSame := true
		for i := 0; i < len(old); i++ {
			if !new[i].MatchWithPreloaded(old[i]) {
				allTheSame = false
			}
		}
		if allTheSame {
			return
		}
	}

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

	var insertedIndexes []int
	var mutations []queueMutation

	// collect all items to be inserted
	x, y := len(old), len(new)
	for x > 0 || y > 0 {
		if x > 0 && lengths[x][y] == lengths[x-1][y] {
			x--
			mutations = append(mutations, deleteMutation(x))
		} else if y > 0 && lengths[x][y] == lengths[x][y-1] {
			y--
			insertedIndexes = append(insertedIndexes, y)
			mutations = append(mutations, insertMutation(x, new[y]))
		} else if x > 0 && y > 0 {
			x--
			y--
		}
	}

	cancel := playlist.BulkUpdate(context.Background())
	defer cancel()

	for _, m := range mutations {
		if m.item == nil {
			insertMatched := lo.ContainsBy(insertedIndexes, func(i int) bool {
				return new[i].MatchWithPreloaded(old[m.index])
			})
			if insertMatched {
				playlist.PullOutItem(m.index)
			} else {
				playlist.RemoveItem(m.index)
			}
		} else {
			playlist.InsertItem(m.item, m.index)
		}
	}

	if len(old) > 0 && (len(new) == 0 || !new[0].MatchWithPreloaded(old[0])) {
		old[0].CancelPlaying()
	}
}
