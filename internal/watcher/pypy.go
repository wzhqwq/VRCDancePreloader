package watcher

import (
	"encoding/json"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
	"log"
	"regexp"
)

var pypyDanceQueueRegex = regexp.MustCompile(`^\[PyPyDanceQueue] (\[.*])`)
var pypyVideoPlayRegex = regexp.MustCompile(`^\[VRCX] VideoPlay\(PyPyDance\) "(.*)",([.\d]{2,}),([.\d]+)`)

var pypyLastQueue = NewLastValue("")
var pypyLastPlayedTime = NewLastValue("")
var pypyLastPlayedURL = NewLastValue("")

func parsePyPyQueue(data []byte) ([]queue.PyPyQueueItem, error) {
	var items []queue.PyPyQueueItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func checkPyPyLine(version int32, prefix []byte, content []byte) bool {
	// [PyPyDanceQueue] [{
	matches := pypyDanceQueueRegex.FindSubmatch(content)
	if len(matches) > 1 {
		pypyLastQueue.Set(version, string(matches[1]))
		return true
	}

	// [VRCX] VideoPlay(PyPyDance) "http://jd.pypy.moe/api/v1/videos/3338.mp4",220,220
	matches = pypyVideoPlayRegex.FindSubmatch(content)
	if len(matches) > 3 {
		pypyLastPlayedURL.Set(version, string(matches[1]))
		pypyLastPlayedTime.Set(version, getTimeStampWithOffset(prefix, matches[2]))
		return true
	}

	return false
}

func forceClearPypyState(version int32) {
	pypyLastQueue.Set(version, "")
	pypyLastPlayedURL.Set(version, "")
	pypyLastPlayedTime.Set(version, "")
}
func forceResetPypyState() {
	pypyLastQueue.Reset("")
	pypyLastPlayedURL.Reset("")
	pypyLastPlayedTime.Reset("")
}
func pypyBacktraceDone() bool {
	return pypyLastQueue.Get() != "" && pypyLastPlayedURL.Get() != ""
}

func pypyPostProcess() {
	lastQueue := pypyLastQueue.Get()
	pypyLastQueue.Reset("")

	if lastQueue != "" {
		// clear the received logs

		// process the last log
		log.Println("Processing queue:\n" + lastQueue)

		var newQueue []queue.QueueItem

		q, err := parsePyPyQueue([]byte(lastQueue))
		if err != nil {
			log.Println("Error processing queue log:")
			log.Println(err)
			return
		}
		for _, i := range q {
			newQueue = append(newQueue, &i)
		}

		diffQueues(playlist.GetQueue(), newQueue)
	}

	lastPlayedURL := pypyLastPlayedURL.Get()
	lastPlayedTime := pypyLastPlayedTime.Get()
	pypyLastPlayedURL.ResetVersion()
	pypyLastPlayedTime.ResetVersion()

	if lastPlayedURL != "" && lastPlayedTime != "" {
		if markURLPlaying(lastPlayedTime, lastPlayedURL) {
			pypyLastPlayedURL.Reset("")
			pypyLastPlayedTime.Reset("")
		}
	}
}
