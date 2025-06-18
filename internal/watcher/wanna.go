package watcher

import (
	"encoding/json"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var wannaLastQueue = ""
var wannaLastHistory = ""
var wannaLastUserData = ""

type wannaUserData struct {
	//Version     string        `json:"version"`
	SongID int `json:"songId"`
	//IsRandom    bool          `json:"isRandom"`
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
		wannaLastHistory = ""
		if len(lastEnteredRoom) > 0 && lastEnteredRoom[0] != '*' {
			lastEnteredRoom = "*" + lastEnteredRoom
		}
		return true
	}

	// history ids deserialized: 1460,1226,1457,5907,1221,1249,10247,5882,5793,1242</color>
	// history ids serialized: 1460,1226,1457,5907,1221,1249,10247,5882,5793,1242</color>
	matches = regexp.MustCompile(`history ids (?:deserialized|serialized): (.*)</color>`).FindSubmatch(line)
	if len(matches) > 1 {
		content := string(matches[1])
		if content == "" {
			wannaLastHistory = "empty"
		} else {
			wannaLastHistory = strings.Split(content, ",")[0]
		}
		return true
	}

	// userData = {
	matches = regexp.MustCompile(`userData = (\{.*})`).FindSubmatch(line)
	if len(matches) > 1 {
		wannaLastUserData = string(matches[1])
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

		playTimeMap[lastPlayedURL] = nowFloat

		return true
	}

	return false
}

func wannaPostProcess() {
	if wannaLastQueue != "" && wannaLastHistory != "" && wannaLastUserData != "" {
		// process the last log
		log.Println("Processing queue:\n" + wannaLastQueue)

		var newQueue []queue.QueueItem

		q, err := parseWannaQueue([]byte(wannaLastQueue))
		if err != nil {
			log.Println("Error processing queue log:")
			log.Println(err)
			return
		}
		if wannaLastHistory != "empty" {
			id, err := strconv.Atoi(wannaLastHistory)
			if err != nil {
				log.Println("Error processing WannaDance history log:")
				log.Println(err)
			}

			var userData wannaUserData
			err = json.Unmarshal([]byte(wannaLastUserData), &userData)
			if err != nil {
				log.Println("Error processing WannaDance user data log:")
				log.Println(err)
			}

			if id != userData.SongID {
				log.Println("User data is not matched with the first song in the WannaDance history")
				return
			}

			newQueue = append(newQueue, &queue.WannaQueueItem{
				SongID:      id,
				Title:       userData.VideoTitle,
				PlayerNames: []string{userData.PlayerName},
			})
		}

		for _, i := range q {
			newQueue = append(newQueue, &i)
		}

		diffQueues(playlist.GetQueue(), newQueue)

		// clear the received logs
		wannaLastQueue = ""
		wannaLastHistory = ""
		wannaLastUserData = ""
	}
}
