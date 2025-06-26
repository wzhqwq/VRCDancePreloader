package watcher

import (
	"bytes"
	"io"
	"math"
	"os"
	"sync"
)

var globalVersion int32

type Line struct {
	version int32
	line    []byte
}

func ReadFromEnd(file *os.File) (int64, error) {
	seekStart, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	globalVersion = math.MaxInt32
	initializeBacktrace()

	err = readReverse(4, file, seekStart)
	if err != nil {
		return 0, err
	}

	return seekStart, nil
}

func ReadNewLines(file *os.File, seekStart int64) (int64, error) {
	_, err := file.Seek(seekStart, io.SeekStart)
	if err != nil {
		return 0, err
	}

	globalVersion = 0

	seekStart += read(4, file)

	postProcess()

	return seekStart, nil
}

func read(wc int, file *os.File) int64 {
	lineChan := make(chan Line, 10000)
	readBytes := int64(0)

	var wg sync.WaitGroup
	for i := 0; i < wc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				processLine(line.version, line.line)
			}
		}()
	}

	const bufSize = 32 * 1024
	buf := make([]byte, bufSize)
	rest := make([]byte, 0, bufSize)

	for {
		n, err := file.Read(buf)
		readBytes += int64(n)
		if n == 0 && err != nil {
			break
		}
		data := append(rest, buf[:n]...)
		start := 0
		for {
			idx := bytes.IndexByte(data[start:], '\n')
			if idx < 0 {
				break
			}
			end := start + idx
			line := data[start:end]
			lineCopy := append([]byte(nil), line...)

			globalVersion++
			lineChan <- Line{globalVersion, lineCopy}

			start = end + 1
		}
		if start < len(data) {
			rest = append(rest[:0], data[start:]...)
		} else {
			rest = rest[:0]
		}
	}
	close(lineChan)

	wg.Wait()

	return readBytes
}

func readReverse(wc int, file *os.File, offset int64) error {
	const bufSize = 32 * 1024
	buf := make([]byte, bufSize)
	rest := make([]byte, 0, bufSize)

	for {
		lineChan := make(chan Line, 10000)

		var wg sync.WaitGroup
		for i := 0; i < wc; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for line := range lineChan {
					backtraceLine(line.version, line.line)
				}
			}()
		}

		if offset == 0 {
			return nil
		}
		offset -= bufSize
		if offset < 0 {
			offset = 0
		}
		_, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			return err
		}

		n, err := file.Read(buf)
		if n == 0 && err != nil {
			return err
		}
		data := append(buf[:n], rest...)
		end := len(data)
		for {
			idx := bytes.LastIndexByte(data[:end], '\n')
			if idx < 0 {
				break
			}
			start := idx + 1
			line := data[start:end]
			if len(line) > 0 {
				lineCopy := append([]byte(nil), line...)

				globalVersion--
				lineChan <- Line{globalVersion, lineCopy}
			}
			end = idx
		}
		if end > 0 {
			rest = append(rest[:0], data[:end]...)
		} else {
			rest = rest[:0]
		}
		close(lineChan)

		wg.Wait()

		if checkBacktrace() {
			break
		}
	}
	return nil
}
