package utils

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func PrettyByteSize(b int64) string {
	return PrettyByteSizeF(float64(b))
}

func PrettyByteSizeF(b float64) string {
	for _, unit := range []string{"", "K", "M", "G", "T", "P", "E", "Z"} {
		if math.Abs(b) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", b, unit)
		}
		b /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", b)
}

func PrettyTime(s time.Duration) string {
	s = max(0, s)
	minutes := s / time.Minute
	seconds := (s - minutes*time.Minute) / time.Second
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func FirstLine(s string) string {
	lines := strings.Split(s, "\n")
	return lines[0]
}
