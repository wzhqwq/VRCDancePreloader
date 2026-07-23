package watcher

import (
	"bytes"
	"time"
)

var backtraceLastTimeStamp *LastValue[string]
var backtraceRoomLogNeeded bool

func initializeBacktrace() {
	backtraceRoomLogNeeded = true
	backtraceLastTimeStamp = NewLastValue("")
}

func backtraceLine(version int32, line []byte) {
	firstMinusIndex := bytes.IndexByte(line, '-')
	if firstMinusIndex == -1 {
		return
	}
	content := bytes.TrimSpace(line[firstMinusIndex+1:])
	prefix := line[:firstMinusIndex]

	if shortExitRegex.Match(content) {
		return
	}

	if checkBehaviourLine(version, content, true) {
		return
	}
	if backtraceRoomLogNeeded {
		if checkPyPyLine(version, prefix, content) {
			backtraceLastTimeStamp.Set(version, getTimeStamp(prefix))
			return
		}
		if checkWannaLine(version, prefix, content) {
			backtraceLastTimeStamp.Set(version, getTimeStamp(prefix))
			return
		}
		if checkDuDuLine(version, prefix, content) {
			backtraceLastTimeStamp.Set(version, getTimeStamp(prefix))
			return
		}
	}
}

func postBacktrace() {
	behaviourPostProcess()

	lastTimeStamp := backtraceLastTimeStamp.Get()
	if lastTimeStamp != "" {
		// TODO time zone
		timeStamp, err := time.Parse("2006.01.02 15:04:05 -0700", string(lastTimeStamp)+" +0800")
		if err == nil && time.Since(timeStamp) < 10*time.Minute {
			pypyPostProcess()
			wannaPostProcess()
			duDuPostProcess()
			return
		}
	}
	forceResetPypyState()
	forceResetWannaState()
	forceResetDuDuState()
}

func checkBacktrace() bool {
	if backtraceRoomLogNeeded {
		lastTimeStamp := backtraceLastTimeStamp.Get()
		if lastTimeStamp != "" {
			// TODO time zone
			timeStamp, err := time.Parse("2006.01.02 15:04:05 -0700", string(lastTimeStamp)+" +0800")
			if err == nil && time.Since(timeStamp) > 10*time.Minute {
				backtraceRoomLogNeeded = false
			}
		}
		if pypyBacktraceDone() || wannaBacktraceDone() || duDuBacktraceDone() {
			backtraceRoomLogNeeded = false
		}
	}

	if behaviourBacktraceDone() {
		postBacktrace()
		return true
	}

	return false
}
