package watcher

import (
	"bytes"
	"log"
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
func parseTimeStampWithOffset(pair string) float64 {
	timeStampText := pair[:19]
	// TODO time zone
	timeStamp, err := time.Parse("2006.01.02 15:04:05 -0700", string(timeStampText)+" +0800")
	if err != nil {
		return 0
	}

	offset := pair[20:]
	nowFloat, err := strconv.ParseFloat(offset, 64)
	if err != nil {
		return 0
	}

	return nowFloat + time.Since(timeStamp).Seconds()
}
func markURLPlaying(pair string, url string) {
	now := parseTimeStampWithOffset(pair)
	log.Println("Playing", url, "at", now)
	playlist.MarkURLPlaying(url, now)
}
