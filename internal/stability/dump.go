package stability

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"
)

// dumpAllGoroutines returns pprof goroutine profile (human readable)
func dumpAllGoroutines() string {
	var buf bytes.Buffer
	// Use pprof goroutine profile for rich output (2 == show goroutine stacks)
	_ = pprof.Lookup("goroutine").WriteTo(&buf, 2)
	return buf.String()
}

// Build a full dump combining timestamp, goroutine dump, and registry info.
func buildFullDump() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== DEADLOCK DUMP at %s ===\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("----- GOROUTINE PROFILE -----\n")
	sb.WriteString(dumpAllGoroutines())
	sb.WriteString("\n\n")
	return sb.String()
}

func PanicIfTimeout(problem string) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	timeStr := time.Now().Format("2006-01-02_15-04-05")
	outPath := fmt.Sprintf("dump_%s_%s.txt", problem, timeStr)

	go func() {
		select {
		case <-ctx.Done():
		case <-time.After(time.Second * 5):
			dump := buildFullDump()
			err := os.WriteFile(outPath, []byte(dump), 0644)
			if err != nil {
				fmt.Printf("failed to write dump to %s: %v\n", outPath, err)
			}
			panic(errors.New("deadlock detected while quitting, dump written to " + outPath))
		}
	}()

	return cancel
}
