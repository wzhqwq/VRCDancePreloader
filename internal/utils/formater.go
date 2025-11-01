package utils

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func PrettyByteSize(b int64) string {
	bf := float64(b)
	for _, unit := range []string{"", "K", "M", "G", "T", "P", "E", "Z"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}

func PrettyTime(s time.Duration) string {
	minutes := s / time.Minute
	seconds := (s - minutes*time.Minute) / time.Second
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func FirstLine(s string) string {
	lines := strings.Split(s, "\n")
	return lines[0]
}
