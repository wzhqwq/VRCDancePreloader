package watcher

import (
	"encoding/json"
	"log"
	"regexp"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
)

var duDuQueueInfoRegex = regexp.MustCompile(`(?:Queue data =|Queue info serialized:)\s*(\[.*])`)
var duDuUserDataRegex = regexp.MustCompile(`deserialize video data:\s*(\{.*}|\s)`)

// var duDuVideoStartRegex = regexp.MustCompile(`OnVideoStart: (Started|Paused) video: ([^,]+), since (.+)`)
// var duDuVideoEndRegex = regexp.MustCompile(`OnVideoEnd: Video|Started video load for URL`)

// var duDuVideoSyncRegex = regexp.MustCompile(`Syncing video to ([.\d]+)`)

var duDuLastQueue = NewLastValue("")
var duDuLastUserData = NewLastValue("")
var duDuQueueChanged = NewLastValue(false)

// var duDuLastPlayedURL = NewLastValue("")
// var duDuLastSyncTime = NewLastValue("")

type duDuUserData struct {
	// Version       string  `json:"ver"`
	ID      int  `json:"id"`
	Shuffle bool `json:"shuffle"`
	// Info      string  `json:"info"`
	User string `json:"user"`
	URL  string `json:"url"`
	// Flip      bool    `json:"flip"`
	// Volume    float64 `json:"volume"`
	Group     string  `json:"group"`
	StartTime float64 `json:"starttime"`
	Dancer    string  `json:"dancer"`
	Artist    string  `json:"artist"`
	Title     string  `json:"title"`
}

func parseDuDuQueue(data []byte) ([]queue.DuDuQueueItem, error) {
	var items []queue.DuDuQueueItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func checkDuDuLine(version int32, prefix []byte, content []byte) bool {
	matches := duDuQueueInfoRegex.FindSubmatch(content)
	if len(matches) > 1 {
		duDuLastQueue.Set(version, string(matches[1]))
		duDuQueueChanged.Set(version, true)
		return true
	}

	matches = duDuUserDataRegex.FindSubmatch(content)
	if len(matches) > 1 {
		duDuLastUserData.Set(version, string(matches[1]))
		duDuQueueChanged.Set(version, true)
		return true
	}

	return false
}

func forceClearDuDuState(version int32) {
	duDuLastQueue.Set(version, "")
	duDuLastUserData.Set(version, "")
	// duDuLastPlayedURL.Set(version, "")
	// duDuLastSyncTime.Set(version, "")
	duDuQueueChanged.Set(version, false)
}
func forceResetDuDuState() {
	duDuLastQueue.Reset("")
	duDuLastUserData.Reset("")
	// duDuLastPlayedURL.Reset("")
	// duDuLastSyncTime.Reset("")
	duDuQueueChanged.Reset(false)
}
func duDuBacktraceDone() bool {
	// return duDuLastQueue.Get() != "" && duDuLastUserData.Get() != "" &&
	// 	duDuLastPlayedURL.Get() != "" && duDuLastSyncTime.Get() != ""
	return duDuLastQueue.Get() != "" && duDuLastUserData.Get() != ""
}

func duDuPostProcess() {
	queueChanged := duDuQueueChanged.Get()
	duDuQueueChanged.Reset(false)

	lastQueue := duDuLastQueue.Get()
	lastUserData := duDuLastUserData.Get()
	duDuLastQueue.ResetVersion()
	duDuLastUserData.ResetVersion()

	if lastQueue != "" && lastUserData != "" {
		if queueChanged {
			log.Println("Unstable queue log, wait for one second.")
		} else {
			duDuLastQueue.Reset("")
			duDuLastUserData.Reset("")
			// process the last log
			log.Println("Processing queue:\n" + lastQueue)

			var newQueue []queue.QueueItem

			q, err := parseDuDuQueue([]byte(lastQueue))
			if err != nil {
				log.Println("Error processing queue log:")
				log.Println(err)
				return
			}
			if lastUserData != "DuDu Dance" {
				log.Println("And user data:\n" + lastUserData)
				var userData duDuUserData
				err = json.Unmarshal([]byte(lastUserData), &userData)
				if err != nil {
					log.Println("Error processing DuDuDance user data log:")
					log.Println(err)
				}

				newQueue = append(newQueue, &queue.DuDuQueueItem{
					SongID:     userData.ID,
					Title:      userData.Title,
					Group:      userData.Group,
					PlayerName: userData.User,
					Random:     userData.Shuffle,
					// The only thing we can't get is the duration
					//Duration:    0,
				})
			}

			for _, i := range q {
				newQueue = append(newQueue, &i)
			}

			diffQueues(playlist.GetQueue(), newQueue)
		}
	}

	// lastPlayedURL := duDuLastPlayedURL.Get()
	// lastSyncTime := duDuLastSyncTime.Get()
	// duDuLastSyncTime.ResetVersion()
	// duDuLastPlayedURL.ResetVersion()

	// if lastPlayedURL != "" && lastSyncTime != "" {
	// 	if markURLPlaying(lastSyncTime, lastPlayedURL) {
	// 		duDuLastSyncTime.Reset("")
	// 	}
	// }
}
