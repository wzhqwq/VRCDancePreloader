package cache_fs

import (
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/samber/lo"
)

type legacyFileLocator struct {
	basePath string
}

func newLegacyFileLocator(path string) *legacyFileLocator {
	return &legacyFileLocator{
		basePath: path,
	}
}

func (l *legacyFileLocator) Scan() ([]DirEntry, error) {
	f, err := os.Open(l.basePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result []os.DirEntry

	batchSize := 1000
	for {
		entries, err := f.ReadDir(batchSize)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		result = append(result, entries...)
	}

	return lo.FilterMap(result, func(entry os.DirEntry, _ int) (DirEntry, bool) {
		name := entry.Name()
		if strings.HasPrefix(name, "etag$") {
			return DirEntry{}, false
		}
		info, err := entry.Info()
		if err != nil {
			return DirEntry{}, false
		}
		var created time.Time
		if attr, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
			created = time.Unix(0, attr.CreationTime.Nanoseconds())
		}

		return DirEntry{
			Name:    "video$" + name,
			Size:    info.Size(),
			Created: created,
		}, true
	}), nil
}

func (l *legacyFileLocator) GetPath(name string) string {
	legacyName := strings.TrimPrefix(name, "video$")
	legacyName = strings.Replace(legacyName, ".vrcdp", ".mp4.vrcdp", 1)
	return path.Join(l.basePath, legacyName)
}
