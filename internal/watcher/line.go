package watcher

import (
	"bytes"
	"regexp"
	"strconv"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
)

var timeStampRegex = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2}`)

var shortExitRegex = regexp.MustCompile(`^\[(?:VRCXC|Vo|A)`)

func postProcess() {
	behaviourPostProcess()
	pypyPostProcess()
	wannaPostProcess()
	pwiPostProcess()
	duDuPostProcess()
}

func processLine(version int32, line []byte) {
	firstMinusIndex := bytes.IndexByte(line, '-')
	if firstMinusIndex == -1 {
		return
	}
	content := bytes.TrimSpace(line[firstMinusIndex+1:])
	prefix := line[:firstMinusIndex]

	if shortExitRegex.Match(content) {
		return
	}

	if checkBehaviourLine(version, content, false) {
		return
	}
	if checkPyPyLine(version, prefix, content) {
		return
	}
	if checkWannaLine(version, prefix, content) {
		return
	}
	if checkDuDuLine(version, prefix, content) {
		return
	}

	checkPWILine(version, content)
}

func getTimeStamp(prefix []byte) string {
	timeStampText := timeStampRegex.Find(prefix)
	if timeStampText == nil {
		return ""
	}
	return string(timeStampText)
}

func getTimeStampWithOffset(prefix []byte, offset []byte) string {
	timeStampText := timeStampRegex.Find(prefix)
	if timeStampText == nil {
		return ""
	}
	return string(timeStampText) + "-" + string(offset)
}
func parseTimeStampWithOffset(pair string) time.Duration {
	timeStampText := pair[:19]
	// TODO time zone
	logTime, err := time.Parse("2006.01.02 15:04:05 -0700", string(timeStampText)+" +0800")
	if err != nil {
		return 0
	}

	offset := pair[20:]
	syncSecond, err := strconv.ParseFloat(offset, 64)
	if err != nil {
		return 0
	}

	return time.Duration(syncSecond*float64(time.Second)) + time.Since(logTime)
}
func markURLPlaying(pair string, url string) bool {
	now := parseTimeStampWithOffset(pair)
	if playlist.MarkURLPlaying(url, now) {
		logger.InfoLn("Confirmed", url, "is playing from", now)
		return true
	}
	return false
}
