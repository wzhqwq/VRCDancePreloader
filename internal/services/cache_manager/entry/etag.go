package entry

import (
	"io"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type Etag struct {
	id      string
	etag    string
	logger  utils.LoggerImpl
	cacheFs *cache_fs.CacheFS
}

func (e *Etag) Read() string {
	if e.etag == "" {
		file, ok := e.cacheFs.GetRO("etag$" + e.id + ".txt")
		if ok {
			// read file as string
			b, err := io.ReadAll(file)
			if err != nil {
				e.logger.ErrorLn("Failed to read etag file:", err)
			} else {
				e.etag = string(b)
			}

			file.Close()
		}
	}
	return e.etag
}

func (e *Etag) Set(etag string) {
	e.etag = etag

	file, err := e.cacheFs.Create("etag$" + e.id + ".txt")
	if err != nil {
		e.logger.ErrorLn("Failed to create etag file:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(etag)
	if err != nil {
		e.logger.ErrorLn("Failed to write etag file:", err)
		return
	}
}

func (e *Etag) GetSize() int64 {
	f, ok := e.cacheFs.GetRO("etag$" + e.id + ".txt")
	if ok {
		stat, err := f.Stat()
		if err == nil {
			return stat.Size()
		}
	}
	return 0
}
