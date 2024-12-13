package utils

import (
	"fmt"
	"runtime"
	"sync"
)

type LoggingMutex struct {
	mu sync.Mutex
}

func (lm *LoggingMutex) Lock() {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		fmt.Printf("Locking at %s:%d\n", file, line)
	}
	lm.mu.Lock()
}

func (lm *LoggingMutex) Unlock() {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		fmt.Printf("Unlocking at %s:%d\n", file, line)
	}
	lm.mu.Unlock()
}
