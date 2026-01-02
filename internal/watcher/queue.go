package watcher

import (
	"slices"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type candidateList struct {
	indices []int
}

func (c *candidateList) add(index int) {
	c.indices = append(c.indices, index)
}
func (c *candidateList) pullOutMatched(predicate func(index int) bool) (int, bool) {
	index, indexInList, matched := lo.FindIndexOf(c.indices, predicate)
	if matched {
		c.indices = slices.Delete(c.indices, indexInList, indexInList+1)
		return index, true
	}
	return 0, false
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

	logger.InfoLn(
		"playlist changed:",
		lo.Map(old, func(item *song.PreloadedSong, _ int) string {
			return item.GetSongId()
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
	// otherwise we fine-tune the playlist

	var replacementCandidates candidateList
	var deletions []int
	reused := make([]*song.PreloadedSong, len(new))

	// collect all items to be created or be deleted
	x, y := len(old), len(new)
	for x > 0 || y > 0 {
		if x > 0 && lengths[x][y] == lengths[x-1][y] {
			x--
			// record the removing mutation and call RemoveFromList if it can't be reused anymore
			deletions = append(deletions, x)
		} else if y > 0 && lengths[x][y] == lengths[x][y-1] {
			y--
			// record the candidate spaces for the reused items
			replacementCandidates.add(y)
		} else if x > 0 && y > 0 {
			x--
			y--
			// it's the same part, we can simply reuse it
			reused[y] = old[x]
		}
	}

	// check same song replacement
	if len(deletions) > 0 {
		if deletions[0] == 1 && old[0].Match(old[1]) {
			deletions[0] = 0
		}
	}

	// we first look at all the deletions
	for _, index := range deletions {
		// find a proper position to put on
		dst, ok := replacementCandidates.pullOutMatched(func(i int) bool {
			return new[i].MatchWithPreloaded(old[index])
		})
		if ok {
			// reuse it directly if we find one position
			reused[dst] = old[index]
		} else {
			// otherwise terminate the lifecycle of this song
			old[index].RemoveFromList()
		}
	}

	newList := lo.Map(reused, func(item *song.PreloadedSong, index int) *song.PreloadedSong {
		if item != nil {
			return item
		}
		// create if empty
		return playlist.CreateFromQueueItem(new[index])
	})

	playlist.UpdateQueue(newList)

	if len(old) > 0 && (len(new) == 0 || !new[0].MatchWithPreloaded(old[0])) {
		old[0].CancelPlaying()
	}
}
