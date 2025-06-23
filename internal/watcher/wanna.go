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

var wannaLastQueue string
var wannaLastUserData string
var wannaQueueChanged bool
var wannaLastPlayedURL string

type wannaUserData struct {
	//Version     string        `json:"version"`
	SongID   int  `json:"songId"`
	IsRandom bool `json:"isRandom"`
	//InfoString  string        `json:"infoString"`
	PlayerName string `json:"playerName"`
	VideoURL   string `json:"videoUrl"`
	VideoTitle string `json:"videoTitle"`
	//Flip        bool          `json:"flip"`
	//Volume      float64       `json:"volume"`
	//DoubleWidth bool          `json:"doubleWidth"`
	//VideoGroup  string        `json:"videoGroup"`
	//VideoMajor  string        `json:"videoMajor"`
	//PlayerCount string        `json:"playerCount"`
	//Motion      []interface{} `json:"motion"`
	//Rpe         int           `json:"rpe"`
}

func parseWannaQueue(data []byte) ([]queue.WannaQueueItem, error) {
	var items []queue.WannaQueueItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func checkWannaLine(line []byte, timeStamp time.Time) bool {
	// syncedQueuedInfoJson = [{
	// queue info serialized: [{
	matches := regexp.MustCompile(`(?:syncedQueuedInfoJson =|queue info serialized:) (\[.*])`).FindSubmatch(line)
	if len(matches) > 1 {
		wannaLastQueue = string(matches[1])
		wannaQueueChanged = true
		return true
	}

	// userData = {
	// userData = Wanna Dance
	matches = regexp.MustCompile(`userData = (\{.*})`).FindSubmatch(line)
	if len(matches) > 1 {
		wannaLastUserData = string(matches[1])
		wannaQueueChanged = true
		return true
	}

	// Started video load for URL: http://api.udon.dance/Api/Songs/play?id=3919, requested by
	// It's always before "Syncing video to 12.37"
	matches = regexp.MustCompile(`Started video load for URL: ([^,]+)`).FindSubmatch(line)
	if len(matches) > 1 {
		wannaLastPlayedURL = string(matches[1])
		return true
	}

	// Syncing video to 12.37
	matches = regexp.MustCompile(`Syncing video to ([.\d]+)`).FindSubmatch(line)
	if len(matches) > 1 {
		now := string(matches[1])

		nowFloat, err := strconv.ParseFloat(now, 64)
		if err != nil {
			return false
		}

		nowFloat += time.Since(timeStamp).Seconds()

		playTimeMap[wannaLastPlayedURL] = nowFloat

		return true
	}

	return false
}

func resetWannaLog() {
	wannaLastQueue = ""
	wannaLastUserData = ""
}

func wannaPostProcess() {
	if wannaLastQueue != "" {
		if wannaQueueChanged {
			wannaQueueChanged = false
			log.Println("Unstable queue log, wait for one second.")
			return
		}
		// process the last log
		log.Println("Processing queue:\n" + wannaLastQueue)

		var newQueue []queue.QueueItem

		q, err := parseWannaQueue([]byte(wannaLastQueue))
		if err != nil {
			log.Println("Error processing queue log:")
			log.Println(err)
			return
		}
		if wannaLastUserData != "" {
			log.Println("And user data:\n" + wannaLastUserData)
			var userData wannaUserData
			err = json.Unmarshal([]byte(wannaLastUserData), &userData)
			if err != nil {
				log.Println("Error processing WannaDance user data log:")
				log.Println(err)
			}

			newQueue = append(newQueue, &queue.WannaQueueItem{
				SongID:      userData.SongID,
				Title:       userData.VideoTitle,
				PlayerNames: []string{userData.PlayerName},
				Random:      userData.IsRandom,
			})
		}

		for _, i := range q {
			newQueue = append(newQueue, &i)
		}

		diffQueues(playlist.GetQueue(), newQueue)
		if len(lastEnteredRoom) > 0 && lastEnteredRoom[0] != '*' {
			lastEnteredRoom = "*" + lastEnteredRoom
		}

		// clear the received logs
		resetWannaLog()
	}
}
