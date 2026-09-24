package hijack

import (
	"io"
	"time"
)

const (
	rateLimitChunkSize = 128 * 1024
	rateLimitDelay     = 50 * time.Millisecond
)

func RateLimitedCopy(dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, rateLimitChunkSize)
	var total int64

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			time.Sleep(rateLimitDelay)
		}
		if readErr != nil {
			if readErr == io.EOF {
				return total, nil
			}
			return total, readErr
		}
	}
}
