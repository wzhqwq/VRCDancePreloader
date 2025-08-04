package fragmented

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// 16KB per trunk
const bytesPerTrunk = 1024 * 16

// 1GB capacity
const capacity = 1024 * 1024 * 1024

const magic = "VRCDP_FRAGMENT"
const magicLen = int64(len(magic))
const magicOffset = int64(0)

// all trunks in the header takes 64KB, it's alright
const numTrunks = capacity / bytesPerTrunk
const trunksOffset = magicOffset + magicLen

// time int64
const lastModifiedLen = 8
const lastModifiedOffset = trunksOffset + numTrunks

const bodyOffset = lastModifiedOffset + lastModifiedLen

type trunkFile struct {
	file         *os.File
	trunks       []byte
	lastModified time.Time

	readerWg *sync.WaitGroup
}

func newTrunkFile(baseName string) *trunkFile {
	dlf, err := os.OpenFile(baseName+".dlt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Failed to open trunk file: %v", err)
		return nil
	}

	f := &trunkFile{
		file:         dlf,
		trunks:       make([]byte, numTrunks),
		lastModified: time.Unix(0, 0),

		readerWg: &sync.WaitGroup{},
	}
	if !f.tryRead() {
		err = dlf.Truncate(bodyOffset)
		if err != nil {
			log.Printf("Failed to truncate trunk file: %v", err)
			return nil
		}

		// write magic
		_, err = dlf.WriteAt([]byte(magic), magicOffset)
		if err != nil {
			log.Printf("Failed to write magic: %v", err)
			return nil
		}

		// write trunks
		for i := 0; i < numTrunks; i++ {
			f.trunks[i] = 0
		}
		_, err = dlf.WriteAt(f.trunks, trunksOffset)
		if err != nil {
			log.Printf("Failed to write trunks: %v", err)
			return nil
		}

		// write last modified time
		int64Buf := make([]byte, lastModifiedLen)
		binary.LittleEndian.PutUint64(int64Buf, 0)
		_, err = dlf.WriteAt(int64Buf, lastModifiedOffset)
		if err != nil {
			log.Printf("Failed to write last modified offset: %v", err)
			return nil
		}
	}
	return f
}

func (t *trunkFile) tryRead() bool {
	size, err := t.file.Seek(0, io.SeekEnd)
	if err != nil {
		return false
	}

	// size check
	if size <= bodyOffset {
		return false
	}

	// magic check
	magicTest := make([]byte, magicLen)
	_, err = t.file.ReadAt(magicTest, 0)
	if err != nil {
		return false
	}
	if string(magicTest) != magic {
		return false
	}

	// read trunks
	_, err = t.file.ReadAt(t.trunks, trunksOffset)
	if err != nil {
		return false
	}

	// read last modified time
	int64Buf := make([]byte, lastModifiedLen)
	_, err = t.file.ReadAt(int64Buf, lastModifiedOffset)
	if err != nil {
		return false
	}
	t.lastModified = time.Unix(int64(binary.LittleEndian.Uint64(int64Buf)), 0)

	return true
}

func trunkRangeToFragment(start, length int) *Fragment {
	return &Fragment{
		start:  int64(start) * bytesPerTrunk,
		length: int64(length) * bytesPerTrunk,
	}
}

func (t *trunkFile) ToFragments() []*Fragment {
	fragments := make([]*Fragment, 0, len(t.trunks))
	startIndex := -1
	for i, b := range t.trunks {
		if b == 0 {
			if startIndex != -1 {
				fragments = append(fragments, trunkRangeToFragment(startIndex, i-startIndex))
				startIndex = -1
			}
			continue
		} else {
			if startIndex == -1 {
				startIndex = i
			}
		}
	}
	if startIndex != -1 {
		fragments = append(fragments, trunkRangeToFragment(startIndex, len(t.trunks)-startIndex))
	}
	if len(fragments) == 0 {
		return []*Fragment{
			newFragment(0, 0),
		}
	}
	return fragments
}

func (t *trunkFile) AppendTo(frag *Fragment, data []byte) error {
	offset := bodyOffset + frag.End()

	n, err := t.file.WriteAt(data, offset)
	if err != nil {
		return err
	}

	frag.length += int64(n)

	err = t.FillTrunks(frag)
	if err != nil {
		return err
	}

	return nil
}

func (t *trunkFile) FillTrunks(frag *Fragment) error {
	fillStart := (frag.start + bytesPerTrunk - 1) / bytesPerTrunk
	fillEnd := frag.End() / bytesPerTrunk
	trunksChanged := false

	for i := fillStart; i < fillEnd; i++ {
		if t.trunks[i] == 0 {
			trunksChanged = true
			t.trunks[i] = 1
		}
	}

	if trunksChanged {
		_, err := t.file.WriteAt(t.trunks, trunksOffset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *trunkFile) ReadAt(p []byte, off int64) (n int, err error) {
	t.readerWg.Add(1)
	defer t.readerWg.Done()

	offset := bodyOffset + off
	return t.file.ReadAt(p, offset)
}

func (t *trunkFile) Close() error {
	return t.file.Close()
}

func (t *trunkFile) SaveAs(filename string) error {
	_, err := t.file.Seek(bodyOffset, io.SeekStart)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, t.file)
	if err != nil {
		return err
	}

	return nil
}

func (t *trunkFile) ScheduleRemove() {
	t.readerWg.Wait()
	err := t.file.Close()
	if err != nil {
		log.Printf("Failed to close trunk file: %v", err)
	}
	err = os.Remove(t.file.Name())
	if err != nil {
		log.Printf("Failed to remove trunk file: %v", err)
	}
}

func (t *trunkFile) ModTime() time.Time {
	return t.lastModified
}

func (t *trunkFile) Init(contentLength int64, lastModified time.Time) {
	err := t.file.Truncate(bodyOffset + contentLength)
	if err != nil {
		log.Printf("Failed to truncate trunk file: %v", err)
	}

	t.lastModified = lastModified
	int64Buf := make([]byte, lastModifiedLen)
	binary.LittleEndian.PutUint64(int64Buf, uint64(lastModified.Unix()))
	_, err = t.file.WriteAt(int64Buf, lastModifiedOffset)
	if err != nil {
		log.Printf("Failed to write last modified offset: %v", err)
	}
}
