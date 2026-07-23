package cache_fs

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Local Cache")

type CacheFS struct {
	legacy *legacyFileLocator
}

type DirEntry struct {
	Name string
	Size int64

	Created time.Time
}

func New(path string) (*CacheFS, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.Mkdir(path, 0777)
		if err != nil {
			return nil, err
		}
	}

	return &CacheFS{newLegacyFileLocator(path)}, nil
}

func (c *CacheFS) Migrate(newPath string, copy bool) error {
	// TODO: v3
	return nil
}

func (c *CacheFS) Upgrade() error {
	// TODO: v3
	return nil
}

func (c *CacheFS) Scan() ([]DirEntry, error) {
	return c.legacy.Scan()
}

func (c *CacheFS) GetRO(name string) (*os.File, bool) {
	filePath := c.legacy.GetPath(name)

	f, err := os.Open(filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.ErrorLn("Failed to open file:", filePath)
		}
		return nil, false
	}

	return f, true
}

func (c *CacheFS) Get(name string) (*os.File, bool) {
	filePath := c.legacy.GetPath(name)

	f, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.ErrorLn("Failed to open file:", filePath)
		}
		return nil, false
	}

	return f, true
}

func (c *CacheFS) Create(name string) (*os.File, error) {
	return os.Create(c.legacy.GetPath(name))
}

func (c *CacheFS) Rename(from, to string) error {
	return os.Rename(c.legacy.GetPath(from), c.legacy.GetPath(to))
}

func (c *CacheFS) Exists(name string) bool {
	_, err := os.Stat(c.legacy.GetPath(name))
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		panic(err)
	}
	return true
}

func (c *CacheFS) DeleteWithoutExt(baseName string) error {
	name := ""
	if strings.HasPrefix(baseName, "video$") {
		if c.Exists(baseName + ".vrcdp") {
			name = baseName + ".vrcdp"
		} else if c.Exists(baseName + ".mp4") {
			name = baseName + ".mp4"
		} else if c.Exists(baseName + ".mp4.dl") {
			name = baseName + ".mp4.dl"
		}
	}
	if strings.HasPrefix(baseName, "etag$") {
		if c.Exists(baseName + ".txt") {
			name = baseName + ".txt"
		}
	}

	if name != "" {
		return c.Delete(name)
	}

	return nil
}

func (c *CacheFS) Delete(name string) error {
	return os.Remove(c.legacy.GetPath(name))
}
