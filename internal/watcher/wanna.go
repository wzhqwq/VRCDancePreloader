package watcher

import (
	"encoding/json"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
	"log"
	"regexp"
)

var wannaQueueInfoRegex = regexp.MustCompile(`(?:syncedQueuedInfoJson =|queue info serialized:) (\[.*])`)
var wannaUserDataRegex = regexp.MustCompile(`userData = (\{.*}|Wanna Dance)`)
var wannaVideoStartRegex = regexp.MustCompile(`OnVideoStart: Started video: ([^,]+)`)
var wannaVideoEndRegex = regexp.MustCompile(`OnVideoEnd: Video \S+ ended`)
var wannaVideoSyncRegex = regexp.MustCompile(`Syncing video to ([.\d]+)`)

var wannaLastQueue = NewLastValue("")
var wannaLastUserData = NewLastValue("")
var wannaQueueChanged = NewLastValue(false)
var wannaLastPlayedURL = NewLastValue("")
var wannaLastSyncTime = NewLastValue("")

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

func checkWannaLine(version int32, prefix []byte, content []byte) bool {
	// syncedQueuedInfoJson = [{
	// queue info serialized: [{
	matches := wannaQueueInfoRegex.FindSubmatch(content)
	if len(matches) > 1 {
		wannaLastQueue.Set(version, string(matches[1]))
		wannaQueueChanged.Set(version, true)
		return true
	}

	// userData = {
	// userData = Wanna Dance
	matches = wannaUserDataRegex.FindSubmatch(content)
	if len(matches) > 1 {
		wannaLastUserData.Set(version, string(matches[1]))
		wannaQueueChanged.Set(version, true)
		return true
	}

	// OnVideoStart: Started video: http://api.udon.dance/Api/Songs/play?id=5247, since owner is playing
	// It's always before "Syncing video to 12.37"
	// So the old sync time is needed to cleared
	matches = wannaVideoStartRegex.FindSubmatch(content)
	if len(matches) > 1 {
		wannaLastPlayedURL.Set(version, string(matches[1]))
		wannaLastSyncTime.Set(version, "")
		return true
	}

	// OnVideoEnd: Video http://api.udon.dance/Api/Songs/play?id=4604 ended
	// Clear lastPlayedURL to prevent video syncing before the next video which is not started yet
	if wannaVideoEndRegex.Match(content) {
		wannaLastPlayedURL.Set(version, "")
		return true
	}

	// Syncing video to 12.37
	matches = wannaVideoSyncRegex.FindSubmatch(content)
	if len(matches) > 1 {
		wannaLastSyncTime.Set(version, getTimeStampWithOffset(prefix, matches[1]))

		return true
	}

	return false
}

func forceClearWannaState(version int32) {
	wannaLastQueue.Set(version, "")
	wannaLastUserData.Set(version, "")
	wannaLastPlayedURL.Set(version, "")
	wannaLastSyncTime.Set(version, "")
	wannaQueueChanged.Set(version, false)
}
func forceResetWannaState() {
	wannaLastQueue.Reset("")
	wannaLastUserData.Reset("")
	wannaLastPlayedURL.Reset("")
	wannaLastSyncTime.Reset("")
	wannaQueueChanged.Reset(false)
}
func wannaBacktraceDone() bool {
	return wannaLastQueue.Get() != "" && wannaLastUserData.Get() != "" &&
		wannaLastPlayedURL.Get() != "" && wannaLastSyncTime.Get() != ""
}

func wannaPostProcess() {
	queueChanged := wannaQueueChanged.Get()
	wannaQueueChanged.Reset(false)

	lastQueue := wannaLastQueue.Get()
	lastUserData := wannaLastUserData.Get()
	wannaLastQueue.ResetVersion()
	wannaLastUserData.ResetVersion()

	if lastQueue != "" && lastUserData != "" {
		if queueChanged {
			log.Println("Unstable queue log, wait for one second.")
		} else {
			wannaLastQueue.Reset("")
			wannaLastUserData.Reset("")
			// process the last log
			log.Println("Processing queue:\n" + lastQueue)

			var newQueue []queue.QueueItem

			q, err := parseWannaQueue([]byte(lastQueue))
			if err != nil {
				log.Println("Error processing queue log:")
				log.Println(err)
				return
			}
			if lastUserData != "Wanna Dance" {
				log.Println("And user data:\n" + lastUserData)
				var userData wannaUserData
				err = json.Unmarshal([]byte(lastUserData), &userData)
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
		}
	}

	lastPlayedURL := wannaLastPlayedURL.Get()
	lastSyncTime := wannaLastSyncTime.Get()
	wannaLastSyncTime.ResetVersion()
	wannaLastPlayedURL.ResetVersion()

	if lastPlayedURL != "" && lastSyncTime != "" {
		if markURLPlaying(lastSyncTime, lastPlayedURL) {
			wannaLastSyncTime.Reset("")
		}
	}
}
