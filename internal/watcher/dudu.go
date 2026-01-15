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
var duDuUserDataRegex = regexp.MustCompile(`deserialize video data:\s*(\{.*})?</color>`)

var duDuVizVidEventRegex = regexp.MustCompile(`VizVid callback: video (loading|playback) started`)
var duDuVideoCountdownRegex = regexp.MustCompile(`starting countdown display, remaining time = ([.\d]+) seconds`)

var duDuLastQueue = NewLastValue("")
var duDuLastUserData = NewLastValue("")
var duDuQueueChanged = NewLastValue(false)

var duDuLastCountdownPair = NewLastValue("")

var duDuVideoChanged = NewLastValue(false)
var duDuVideoPlaying = NewLastValue(false)

var duduLogger = utils.NewLogger("DuDuFitDance Log Watcher")

type duDuUserData struct {
	// Version       string  `json:"ver"`
	ID      int  `json:"id"`
	Shuffle bool `json:"shuffle"`
	// Info      string  `json:"info"`
	User string `json:"user"`
	URL  string `json:"url"`
	// Flip      bool    `json:"flip"`
	// Volume    float64 `json:"volume"`
	Group string `json:"group"`
	//StartTime float64 `json:"starttime"`
	Dancer string `json:"dancer"`
	Artist string `json:"artist"`
	Title  string `json:"title"`
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
			duDuLastUserData.Set(version, "empty")
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
			// and the previous video must be ended
			duDuVideoPlaying.Set(version, false)
			// user data is now stable
			duDuVideoChanged.Set(version, true)
		} else {
			// playback started
			duDuVideoPlaying.Set(version, true)
		}
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
	duDuVideoPlaying.Set(version, false)
	duDuQueueChanged.Set(version, false)
}
func forceResetDuDuState() {
	duDuLastQueue.Reset("")
	duDuLastUserData.Reset("")
	duDuLastCountdownPair.Reset("")
	duDuVideoChanged.Reset(false)
	duDuVideoPlaying.Reset(false)
	duDuQueueChanged.Reset(false)
}
func duDuBacktraceDone() bool {
	return duDuLastQueue.Get() != "" && duDuLastUserData.Get() != "" && duDuVideoChanged.Get()
}

var userDataDuDu duDuUserData

func duDuPostProcess() {
	queueChanged := duDuQueueChanged.Get()
	duDuQueueChanged.Reset(false)

	lastQueue := duDuLastQueue.Get()
	lastUserData := duDuLastUserData.Get()
	duDuLastQueue.ResetVersion()
	duDuLastUserData.ResetVersion()
	duDuLastCountdownPair.ResetVersion()
	duDuVideoChanged.ResetVersion()
	duDuVideoPlaying.ResetVersion()

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
				err = json.Unmarshal([]byte(lastUserData), &userDataDuDu)
				if err != nil {
					duduLogger.ErrorLn("Error processing DuDuFitDance user data log:", err)
				}

				newQueue = append(newQueue, &queue.DuDuQueueItem{
					SongID:     userDataDuDu.ID,
					Title:      userDataDuDu.Title,
					Group:      userDataDuDu.Group,
					PlayerName: userDataDuDu.User,
					Random:     userDataDuDu.Shuffle,
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

	if !queueChanged && duDuVideoChanged.Get() {
		// user data is now stable
		trySyncDuDuVideo()
	}
}

func trySyncDuDuVideo() {
	// try sync using count down
	if lastCountdownPair := duDuLastCountdownPair.Get(); lastCountdownPair != "" {
		syncDuDuVideoUsingCountdown(lastCountdownPair)
		return
	}

	if duDuVideoPlaying.Get() {
		// well, it's already playing and we cannot sync with it anymore
		justPlayDuDuVideo()
	}
}

func syncDuDuVideoUsingCountdown(pair string) {
	countDown := -parseTimeStampWithOffset(pair, true)
	if countDown > time.Second*30 {
		// invalidate count down
		duDuLastCountdownPair.Reset("")
		return
	}
	if playlist.MarkURLPlaying(userDataDuDu.URL, -countDown) {
		if countDown > 0 {
			duduLogger.InfoLn("Confirmed", userDataDuDu.URL, "will play after", countDown)
		} else {
			duduLogger.InfoLn("Confirmed", userDataDuDu.URL, "is playing from", -countDown)
		}
		duDuLastCountdownPair.Reset("")
		duDuVideoChanged.Reset(false)
	}
}

func justPlayDuDuVideo() {
	if playlist.MarkURLPlayingWithoutSync(userDataDuDu.URL) {
		duduLogger.InfoLn("Confirmed", userDataDuDu.URL, "is playing, but we cannot sync with it")
		duDuVideoPlaying.Reset(false)
		duDuVideoChanged.Reset(false)
	}
}
