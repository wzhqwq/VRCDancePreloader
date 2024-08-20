package watcher

import (
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

var logWatcher *fsnotify.Watcher
var dirWatcher *fsnotify.Watcher
var seekStart int64

func sniffActiveLog(logBase string) (string, error) {
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
		return "", nil
	}

	log.Println("Found active log:", latestFile.Name())

	return logBase + "/" + latestFile.Name(), nil
}

func watchLog(logPath string) error {
	// watch the log file for changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(logPath)
	if err != nil {
		return err
	}

	logWatcher = watcher
	defer func() {
		logWatcher = nil
	}()

	log.Println("Watching log:", logPath)

	seekStart = 0
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}

	seekStart, err = ReadNewLines(file, seekStart)
	if err != nil {
		return err
	}
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				// read the new lines from the log
				// process the new lines
				file, err := os.Open(logPath)
				if err != nil {
					return err
				}

				seekStart, err = ReadNewLines(file, seekStart)
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

func watchDir(logBase string) error {
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

			if event.Op&fsnotify.Create == fsnotify.Create {
				path := event.Name
				info, err := os.Stat(path)
				if err != nil || info.IsDir() || info.Name()[:10] != "output_log" {
					continue
				}

				if logWatcher != nil {
					logWatcher.Close()
				}
				go func() {
					err = watchLog(path)
					if err != nil {
						log.Println(err)
					}
				}()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			return err
		}
	}
}

func Start(logBase string) error {
	// start watching the log directory
	logPath, err := sniffActiveLog(logBase)
	if err != nil {
		return err
	}

	go func() {
		err = watchLog(logPath)
		if err != nil {
			log.Println(err)
		}
	}()

	go func() {
		err = watchDir(logBase)
		if err != nil {
			log.Println(err)
		}
	}()

	return nil
}

func Stop() {
	// stop watching the log directory
	if logWatcher != nil {
		logWatcher.Close()
	}

	if dirWatcher != nil {
		dirWatcher.Close()
	}
}
