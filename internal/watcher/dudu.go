package watcher

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher/queue"
)

var duDuQueueInfoRegex = regexp.MustCompile(`(?:Queue data =|Queue info serialized:)\s*(\[.*])`)
var duDuUserDataRegex = regexp.MustCompile(`deserialize video data:\s*(\{.*})?`)

var duDuVizVidEventRegex = regexp.MustCompile(`VizVid callback: video (loading|playback) started`)
var duDuVideoCountdownRegex = regexp.MustCompile(`starting countdown display, remaining time = (\d+) seconds`)

var duDuLastQueue = NewLastValue("")
var duDuLastUserData = NewLastValue("")
var duDuQueueChanged = NewLastValue(false)

var duDuLastCountdownPair = NewLastValue("")

// var duDuVideoStatus = NewLastValue("")
var duDuVideoChanged = NewLastValue(false)

var duduLogger = utils.NewLogger("DuDuFitDance Log Watcher")

// A magic date (1761955200) used when the room does not use "server time" to sync with the master client
// I don't know why this fixed date is used sometimes (to prevent failed calls to GetServerTimeInSeconds?)
var serverStartTimeFallback = time.Date(2025, time.November, 1, 0, 0, 0, 0, time.UTC)

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
		if len(matches[1]) > 0 {
			duDuLastUserData.Set(version, string(matches[1]))
		} else {
			//duDuLastUserData.Set(version, "empty")
		}
		duDuQueueChanged.Set(version, true)
		return true
	}

	matches = duDuVizVidEventRegex.FindSubmatch(content)
	if len(matches) > 1 {
		event := string(matches[1])
		if event == "loading" {
			// every new count down starts after VizVid starts loading
			// don't leave old countdown record
			duDuLastCountdownPair.Set(version, "")
			duDuVideoChanged.Set(version, true)
		}
		//duDuVideoStatus.Set(version, event)
		return true
	}

	matches = duDuVideoCountdownRegex.FindSubmatch(content)
	if len(matches) > 1 {
		duDuLastCountdownPair.Set(version, getTimeStampWithOffset(prefix, matches[1]))
		return true
	}

	return false
}

func forceClearDuDuState(version int32) {
	duDuLastQueue.Set(version, "")
	duDuLastUserData.Set(version, "")
	duDuLastCountdownPair.Set(version, "")
	duDuVideoChanged.Set(version, false)
	duDuQueueChanged.Set(version, false)
}
func forceResetDuDuState() {
	duDuLastQueue.Reset("")
	duDuLastUserData.Reset("")
	duDuLastCountdownPair.Reset("")
	duDuVideoChanged.Reset(false)
	duDuQueueChanged.Reset(false)
}
func duDuBacktraceDone() bool {
	return duDuLastQueue.Get() != "" && duDuLastUserData.Get() != "" && duDuVideoChanged.Get()
}

func duDuPostProcess() {
	queueChanged := duDuQueueChanged.Get()
	duDuQueueChanged.Reset(false)

	lastQueue := duDuLastQueue.Get()
	lastUserData := duDuLastUserData.Get()
	lastCountdownPair := duDuLastCountdownPair.Get()
	videoChanged := duDuVideoChanged.Get()
	duDuLastQueue.ResetVersion()
	duDuLastUserData.ResetVersion()
	duDuLastCountdownPair.ResetVersion()
	duDuVideoChanged.ResetVersion()
	//duDuVideoStatus.ResetVersion()

	if lastQueue != "" && lastUserData != "" {
		if queueChanged {
			duduLogger.InfoLn("Unstable queue log, wait for one second.")
		} else {
			duDuLastQueue.Reset("")
			duDuLastUserData.Reset("")
			// process the last log
			duduLogger.DebugLn("Processing queue:\n" + lastQueue)

			var newQueue []queue.QueueItem

			q, err := parseDuDuQueue([]byte(lastQueue))
			if err != nil {
				duduLogger.ErrorLn("Error processing queue log:", err)
				return
			}
			if lastUserData != "empty" {
				duduLogger.DebugLn("And user data:\n" + lastUserData)
				var userData duDuUserData
				err = json.Unmarshal([]byte(lastUserData), &userData)
				if err != nil {
					duduLogger.ErrorLn("Error processing DuDuFitDance user data log:", err)
				}

				if videoChanged {
					if syncDuDuVideoUsingStartTime(userData) {
						duDuVideoChanged.Reset(false)
					} else {
						if lastCountdownPair != "" && syncDuDuVideoUsingCountdown(userData, lastCountdownPair) {
							duDuLastCountdownPair.Reset("")
							duDuVideoChanged.Reset(false)
						}
					}
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
}

func syncDuDuVideoUsingStartTime(userData duDuUserData) bool {
	var relativeStart = time.Duration(userData.StartTime*1000) * time.Millisecond
	// video waits a countdown (often 10s) to end before playing
	var estimatedStart = serverStartTimeFallback.Add(relativeStart + time.Second*10)
	var offset = time.Now().Sub(estimatedStart)

	if offset > time.Hour {
		// so it's not a fallback
		// we cannot call GetServerTimeInSeconds (which is inside Udon) to calculate when server starts
		// so unfortunately we cannot sync with the playing video unless the room is using fallback clock
		duDuVideoChanged.Reset(false)
		return false
	}

	if playlist.MarkURLPlaying(userData.URL, offset) {
		duduLogger.InfoLn("Confirmed", userData.URL, "is playing from", offset, "using 'starttime'")
		return true
	}

	return false
}

func syncDuDuVideoUsingCountdown(userData duDuUserData, pair string) bool {
	countDown := parseTimeStampWithOffset(pair, true)
	if countDown > time.Second*30 {
		// return true to invalidate count down
		return true
	}
	if playlist.MarkURLPlaying(userData.URL, -countDown) {
		duduLogger.InfoLn("Confirmed", userData.URL, "will play after", countDown)
		return true
	}
	return false
}
