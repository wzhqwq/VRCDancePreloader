package watcher

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
)

var receivedLogs = make([]string, 0)
var playTimeMap = make(map[string]float64)

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

	if len(receivedLogs) > 0 {
		// process the last log
		lastLog := receivedLogs[len(receivedLogs)-1]
		log.Println("Processing queue log:\n" + lastLog)

		err := processQueueLog([]byte(lastLog))
		if err != nil {
			log.Println("Error processing queue log:")
			log.Println(err)
		}

		// clear the received logs
		receivedLogs = make([]string, 0)
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
	timeStampText := regexp.MustCompile(`\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2}`).Find(line)
	if timeStampText == nil {
		return
	}
	timeStamp, err := time.Parse("2006.01.02 15:04:05 -0700", string(timeStampText)+" +0800")
	if err != nil {
		return
	}
	// only process logs at most 10 minutes ago
	if time.Since(timeStamp) > 10*time.Minute {
		return
	}

	matches := regexp.MustCompile(`\[PyPyDanceQueue\] (\[.*\])$`).FindSubmatch(line)
	if len(matches) > 1 {
		receivedLogs = append(receivedLogs, string(matches[1]))
	}
	// VideoPlay(PyPyDance) "http://jd.pypy.moe/api/v1/videos/3338.mp4",220,220
	matches = regexp.MustCompile(`VideoPlay\(PyPyDance\) "(.*)",([\.\d]+),([\.\d]+)`).FindSubmatch(line)
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
