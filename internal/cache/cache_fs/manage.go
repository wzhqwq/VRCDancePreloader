package cache_fs

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Local Cache")

var legacy *legacyFileLocator

type DirEntry struct {
	Name string
	Size int64

	Created time.Time
}

func Setup(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}

	legacy = newLegacyFileLocator(path)
}

func Migrate(newPath string, copy bool) error {
	// TODO: v3
	return nil
}

func Upgrade() error {
	// TODO: v3
	return nil
}

func Scan() ([]DirEntry, error) {
	return legacy.Scan()
}

func GetRO(name string) (*os.File, bool) {
	filePath := legacy.GetPath(name)

	f, err := os.Open(filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.ErrorLn("Failed to open file:", filePath)
		}
		return nil, false
	}

	return f, true
}

func Get(name string) (*os.File, bool) {
	filePath := legacy.GetPath(name)

	f, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.ErrorLn("Failed to open file:", filePath)
		}
		return nil, false
	}

	return f, true
}

func Create(name string) (*os.File, error) {
	return os.Create(legacy.GetPath(name))
}

func Rename(from, to string) error {
	return os.Rename(legacy.GetPath(from), legacy.GetPath(to))
}

func Exists(name string) bool {
	_, err := os.Stat(legacy.GetPath(name))
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		panic(err)
	}
	return true
}

func DeleteWithoutExt(baseName string) error {
	name := ""
	if strings.HasPrefix(baseName, "video$") {
		if Exists(baseName + ".vrcdp") {
			name = baseName + ".vrcdp"
		} else if Exists(baseName + ".mp4") {
			name = baseName + ".mp4"
		} else if Exists(baseName + ".mp4.dl") {
			name = baseName + ".mp4.dl"
		}
	}
	if strings.HasPrefix(baseName, "etag$") {
		if Exists(baseName + ".txt") {
			name = baseName + ".txt"
		}
	}

	if name != "" {
		return Delete(name)
	}

	return nil
}

func Delete(name string) error {
	return os.Remove(legacy.GetPath(name))
}
