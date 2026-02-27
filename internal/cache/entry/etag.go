package entry

import (
	"io"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/cache_fs"
)

func (e *BaseEntry) readEtag() {
	if e.etag == "" {
		file, ok := cache_fs.GetRO("etag$" + e.id + ".txt")
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
}

func (e *BaseEntry) setEtag(etag string) {
	e.etag = etag

	file, err := cache_fs.Create("etag$" + e.id + ".txt")
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
