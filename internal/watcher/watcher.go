package watcher

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

var dirWatcher *fsnotify.Watcher
var watchingFile *os.File
var logBase string

func sniffActiveLog() (string, error) {
	// vrchat log format: output_log_2024-08-15_21-22-15.txt
	// find the latest log
	var latestFile os.FileInfo
	logDir, err := os.ReadDir(logBase)
	if err != nil {
		return "", err
	}

	for _, log := range logDir {
		if log.IsDir() || len(log.Name()) < 15 || log.Name()[:10] != "output_log" {
			continue
		}

		info, err := log.Info()
		if err != nil {
			continue
		}

		if latestFile == nil || info.ModTime().After(latestFile.ModTime()) {
			latestFile = info
		}
	}

	if latestFile == nil {
		return "", errors.New("no active log found")
	}

	log.Println("Found active log:", latestFile.Name())

	return logBase + "/" + latestFile.Name(), nil
}

func keepTrackUntilClose(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	watchingFile = file

	go func() {
		seekStart := int64(0)
		for {
			seekStart, err = ReadNewLines(file, seekStart)
			if err != nil && err.Error() != "EOF" {
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

	log.Println("Watching directory:", logBase)

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
			log.Panic(err)
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
