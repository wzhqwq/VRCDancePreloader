package watcher

import (
	"errors"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

var dirWatcher *fsnotify.Watcher
var seekStart int64
var nowWatching string
var logBase string

func sniffActiveLog() error {
	// vrchat log format: output_log_2024-08-15_21-22-15.txt
	// find the latest log
	var latestFile os.FileInfo
	logDir, err := os.ReadDir(logBase)
	if err != nil {
		return err
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
		return errors.New("no active log found")
	}

	nowWatching = latestFile.Name()
	log.Println("Found active log:", latestFile.Name())

	return nil
}

func processFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	seekStart, err = ReadNewLines(file, seekStart)
	return err
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
				nowWatching = info.Name()
				seekStart = 0
				if processFile(path) != nil {
					return err
				}
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				if nowWatching != info.Name() {
					continue
				}
				if processFile(path) != nil {
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
	logBase = base
	// start watching the log directory
	err := sniffActiveLog()
	if err != nil {
		return err
	}

	if processFile(logBase+"/"+nowWatching) != nil {
		return err
	}

	go func() {
		err = watch()
		if err != nil {
			log.Println(err)
		}
	}()

	return nil
}

func Stop() {
	// stop watching the log directory
	if dirWatcher != nil {
		dirWatcher.Close()
	}
}
