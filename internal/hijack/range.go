package hijack

import (
	"github.com/samber/lo"
	"net/textproto"
	"strconv"
	"strings"
)

// simplified version from net/http/fs.go
func parseRange(s string, size int64) int64 {
	if s == "" {
		return 0
	}
	if !strings.HasPrefix(s, "bytes=") {
		return 0
	}
	var offsets []int64
	for _, r := range strings.Split(strings.TrimPrefix(s, "bytes="), ",") {
		r = textproto.TrimString(r)
		if r == "" {
			continue
		}
		start, end, ok := strings.Cut(r, "-")
		if !ok {
			return 0
		}
		start, end = textproto.TrimString(start), textproto.TrimString(end)
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file,
			// and we are dealing with <suffix-length>
			// which has to be a non-negative integer as per
			// RFC 7233 Section 2.1 "Byte-Ranges".
			if end == "" || end[0] == '-' {
				return 0
			}
			i, err := strconv.ParseInt(end, 10, 64)
			if i < 0 || err != nil {
				return 0
			}
			if i > size {
				i = size
			}
			offsets = append(offsets, size-i)
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return 0
			}
			if i >= size {
				continue
			}
			offsets = append(offsets, i)
		}
	}

	if len(offsets) == 0 {
		return 0
	}
	return lo.Min(offsets)
}
