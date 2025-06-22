package watcher

import (
	"encoding/json"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
	"log"
	"regexp"
	"strconv"
	"time"
)

var pypyLastQueue string

func parsePyPyQueue(data []byte) ([]queue.PyPyQueueItem, error) {
	var items []queue.PyPyQueueItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func checkPyPyLine(line []byte, timeStamp time.Time) bool {
	// [PyPyDanceQueue] [{
	matches := regexp.MustCompile(`\[PyPyDanceQueue] (\[.*])`).FindSubmatch(line)
	if len(matches) > 1 {
		pypyLastQueue = string(matches[1])
		if len(lastEnteredRoom) > 0 && lastEnteredRoom[0] != '*' {
			lastEnteredRoom = "*" + lastEnteredRoom
		}
		return true
	}

	// VideoPlay(PyPyDance) "http://jd.pypy.moe/api/v1/videos/3338.mp4",220,220
	matches = regexp.MustCompile(`VideoPlay\(PyPyDance\) "(.*)",([.\d]+),([.\d]+)`).FindSubmatch(line)
	if len(matches) > 1 {
		url := string(matches[1])
		now := string(matches[2])
		dur := string(matches[3])

		if now != "0" {
			nowFloat, err := strconv.ParseFloat(now, 64)
			if err != nil {
				return false
			}
			durFloat, err := strconv.ParseFloat(dur, 64)
			if err != nil {
				return false
			}

			nowFloat += time.Since(timeStamp).Seconds()
			if nowFloat > durFloat {
				return false
			}

			playTimeMap[url] = nowFloat
		}

		return true
	}

	return false
}

func pypyPostProcess() {
	if pypyLastQueue != "" {
		// process the last log
		log.Println("Processing queue:\n" + pypyLastQueue)

		var newQueue []queue.QueueItem

		q, err := parsePyPyQueue([]byte(pypyLastQueue))
		if err != nil {
			log.Println("Error processing queue log:")
			log.Println(err)
			return
		}
		for _, i := range q {
			newQueue = append(newQueue, &i)
		}

		diffQueues(playlist.GetQueue(), newQueue)

		// clear the received logs
		pypyLastQueue = ""
	}
}
