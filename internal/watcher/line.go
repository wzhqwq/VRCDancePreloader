package watcher

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/service"
)

var playTimeMap = make(map[string]float64)

var lastQueue = ""
var lastEnteredRoom = ""

func ReadNewLines(file *os.File, seekStart int64) (int64, error) {
	file.Seek(seekStart, 0)

	buf := make([]byte, 1024)
	lineBuf := make([]byte, 0)

	for {
		n, err := file.Read(buf)
		if err != nil {
			break
		}

		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				processLine(lineBuf)
				lineBuf = make([]byte, 0)
			} else {
				lineBuf = append(lineBuf, buf[i])
			}
		}

		seekStart += int64(n)
	}

	if lastQueue != "" {
		// process the last log
		log.Println("Processing queue:\n" + lastQueue)

		err := processQueueLog([]byte(lastQueue))
		if err != nil {
			log.Println("Error processing queue log:")
			log.Println(err)
		}

		// clear the received logs
		lastQueue = ""
	}

	// play time map
	if len(playTimeMap) > 0 {
		for url, now := range playTimeMap {
			log.Println("Playing", url, "at", now)
			playlist.MarkURLPlaying(url, now)
		}
		playTimeMap = make(map[string]float64)
	}

	return seekStart, nil
}

func processLine(line []byte) {
	// [Behaviour] Entering Room: PyPyDance
	matches := regexp.MustCompile(`\[Behaviour] Entering Room: (.*)`).FindSubmatch(line)
	if len(matches) > 1 {
		lastEnteredRoom = string(matches[1])
		return
	}

	// [Behaviour] Joining wrld_f20326da-f1ac-45fc-a062-609723b097b1:29406~region(jp)
	matches = regexp.MustCompile(`\[Behaviour] Joining (wrld_.*):.*`).FindSubmatch(line)
	if len(matches) > 1 {
		service.SetCurrentWorldID(string(matches[1]))
		return
	}

	// time related

	timeStampText := regexp.MustCompile(`\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2}`).Find(line)
	if timeStampText == nil {
		return
	}
	// TODO time zone
	timeStamp, err := time.Parse("2006.01.02 15:04:05 -0700", string(timeStampText)+" +0800")
	if err != nil {
		return
	}
	// only process logs at most 10 minutes ago
	if time.Since(timeStamp) > 10*time.Minute {
		return
	}

	// [PyPyDanceQueue] [{
	matches = regexp.MustCompile(`\[PyPyDanceQueue] (\[.*])`).FindSubmatch(line)
	if len(matches) > 1 {
		lastQueue = string(matches[1])
		if len(lastEnteredRoom) > 0 && lastEnteredRoom[0] != '*' {
			lastEnteredRoom = "*" + lastEnteredRoom
		}
		return
	}

	// [VRCX-World] {
	if service.IsPWIOn() {
		matches = regexp.MustCompile(`\[VRCX-World] (\{.*})`).FindSubmatch(line)
		if len(matches) > 1 {
			err = processPwiLog([]byte(lastEnteredRoom))
			if err != nil {
				log.Println("Error while processing PWI request: " + err.Error())
			}
		}
	}

	// VideoPlay(PyPyDance) "http://jd.pypy.moe/api/v1/videos/3338.mp4",220,220
	matches = regexp.MustCompile(`VideoPlay\(PyPyDance\) "(.*)",([.\d]+),([.\d]+)`).FindSubmatch(line)
	if len(matches) > 1 {
		url := string(matches[1])
		now := string(matches[2])
		dur := string(matches[3])

		nowFloat, err := strconv.ParseFloat(now, 64)
		if err != nil {
			return
		}
		durFloat, err := strconv.ParseFloat(dur, 64)
		if err != nil {
			return
		}

		nowFloat += time.Since(timeStamp).Seconds()
		if nowFloat > durFloat {
			return
		}

		playTimeMap[url] = nowFloat
	}
}
