package watcher

import (
	"errors"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var dirWatcher *fsnotify.Watcher
var watchingFile *os.File
var logBase string

var logger = utils.NewLogger("Log Watcher")

func sniffActiveLog() (string, error) {
	// vrchat entry format: output_log_2024-08-15_21-22-15.txt
	// find the latest entry
	var latestFile os.FileInfo
	logDir, err := os.ReadDir(logBase)
	if err != nil {
		return "", err
	}

	for _, entry := range logDir {
		if entry.IsDir() || len(entry.Name()) < 15 || entry.Name()[:10] != "output_log" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if latestFile == nil || info.ModTime().After(latestFile.ModTime()) {
			latestFile = info
		}
	}

	if latestFile == nil {
		return "", errors.New("no active log file found")
	}

	logger.InfoLn("Found latest active log file:", latestFile.Name())

	return logBase + "/" + latestFile.Name(), nil
}

func keepTrackUntilClose(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	watchingFile = file

	logger.InfoLn("Watching file:", path)

	go func() {
		t := time.Now()
		seekStart, err := ReadFromEnd(file)
		logger.InfoLn("Reading from end takes", time.Since(t).Milliseconds(), "ms")
		if err != nil {
			logger.ErrorLn("Error reading from end:", err)
			return
		}
		for {
			seekStart, err = ReadNewLines(file, seekStart)
			if err != nil {
				return
			}
			<-time.After(1 * time.Second)
		}
	}()

	return nil
}

func watch() error {
	// watch the log directory for new log files
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(logBase)
	if err != nil {
		return err
	}

	dirWatcher = watcher
	defer func() {
		dirWatcher = nil
	}()

	logger.InfoLn("Watching directory:", logBase)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			path := event.Name
			info, err := os.Stat(path)
			if err != nil || info.IsDir() || info.Name()[:10] != "output_log" {
				continue
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				if watchingFile != nil {
					watchingFile.Close()
				}
				err = keepTrackUntilClose(path)
				if err != nil {
					return err
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			return err
		}
	}
}

func Start(base string) error {
	// check if the log directory exists first
	_, err := os.Stat(base)
	if err != nil {
		return err
	}

	logBase = base
	// start watching the log directory
	path, err := sniffActiveLog()
	if err == nil {
		if keepTrackUntilClose(path) != nil {
			return err
		}
	}

	go func() {
		err = watch()
		if err != nil {
			logger.ErrorLn("Error watching directory:", err)
			panic(err)
		}
	}()

	return nil
}

func Stop() {
	// stop watching the log directory
	if dirWatcher != nil {
		dirWatcher.Close()
	}
	if watchingFile != nil {
		watchingFile.Close()
	}
}
