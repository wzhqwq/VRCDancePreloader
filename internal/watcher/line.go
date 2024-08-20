package watcher

import (
	"log"
	"os"
	"regexp"
)

var receivedLogs = make([]string, 0)

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

	return seekStart, nil
}

func processLine(line []byte) {
	matches := regexp.MustCompile(`\[PyPyDanceQueue\] (\[.*\])$`).FindSubmatch(line)
	if len(matches) > 1 {
		receivedLogs = append(receivedLogs, string(matches[1]))
	}
}
