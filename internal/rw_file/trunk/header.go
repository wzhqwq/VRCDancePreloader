package trunk

import (
	"encoding/binary"
	"log"
	"time"
)

// structure:
// | magic | full_size(int64) | last_modified(int64) | states(byte) |
// |                          trunks (16KB)                         |
// |                       body (256MB maximum)                     |

// magic 11 bytes
const magic = "VRCDP_CACHE"
const magicLen = len(magic)
const magicOffset = int64(0)

// full size int64
const fullSizeLen = 8
const fullSizeOffset = magicOffset + int64(magicLen)

// last modified time int64
const lastModifiedLen = 8
const lastModifiedOffset = fullSizeOffset + fullSizeLen

// state byte
const stateCompletedFlag = 0x01
const statesLen = 1
const statesOffset = lastModifiedOffset + lastModifiedLen

// trunks []byte
// all trunks in the header takes 16KB, it's alright
const numTrunks = capacity / bytesPerTrunk
const trunksOffset = statesOffset + statesLen

// body
const bodyOffset = trunksOffset + numTrunks

func (f *File) tryCreate() bool {
	err := f.file.Truncate(bodyOffset)
	if err != nil {
		log.Printf("Failed to truncate cache file: %v", err)
		return false
	}

	// write magic
	_, err = f.file.WriteAt([]byte(magic), magicOffset)
	if err != nil {
		log.Printf("Failed to write magic: %v", err)
		return false
	}

	// write full size
	if !f.writeFullSize() {
		return false
	}

	// write last modified time
	if !f.writeLastModifiedTime() {
		return false
	}

	// write states
	if !f.writeStates() {
		return false
	}

	// write trunks
	if !f.writeTrunks() {
		return false
	}

	return true
}

func (f *File) writeFullSize() bool {
	int64Buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(int64Buf, uint64(f.FullSize))
	_, err := f.file.WriteAt(int64Buf, fullSizeOffset)
	if err != nil {
		log.Printf("Failed to write full size: %v", err)
		return false
	}
	return true
}

func (f *File) writeLastModifiedTime() bool {
	int64Buf := make([]byte, 8)
	unix := f.LastModified.Unix()
	binary.LittleEndian.PutUint64(int64Buf, uint64(unix))
	_, err := f.file.WriteAt(int64Buf, lastModifiedOffset)
	if err != nil {
		log.Printf("Failed to write last modified time: %v", err)
		return false
	}
	return true
}

func (f *File) writeStates() bool {
	stateByte := byte(0)
	if f.Completed {
		stateByte |= stateCompletedFlag
	}

	_, err := f.file.WriteAt([]byte{stateByte}, statesOffset)
	if err != nil {
		log.Printf("Failed to write states: %v", err)
		return false
	}
	return true
}

func (f *File) writeTrunks() bool {
	_, err := f.file.WriteAt(f.trunks, trunksOffset)
	if err != nil {
		log.Printf("Failed to write trunks: %v", err)
		return false
	}
	return true
}

func (f *File) tryRead() bool {
	stat, err := f.file.Stat()
	if err != nil {
		log.Printf("Failed to stat file: %v", err)
		return false
	}

	size := stat.Size()
	if size <= bodyOffset {
		if size > 0 {
			log.Printf("Corrupted file: %s, re-initialize it", f.file.Name())
		}
		return false
	}

	// magic check
	magicTest := make([]byte, magicLen)
	_, err = f.file.ReadAt(magicTest, 0)
	if err != nil {
		log.Printf("Failed to read magic: %v", err)
		return false
	}
	if string(magicTest) != magic {
		log.Printf("Corrupted file: %s, re-initialize it", f.file.Name())
		return false
	}

	int64Buf := make([]byte, lastModifiedLen)

	// read full size
	_, err = f.file.ReadAt(int64Buf, fullSizeOffset)
	if err != nil {
		log.Printf("Failed to read full size: %v", err)
		return false
	}
	f.FullSize = int64(binary.LittleEndian.Uint64(int64Buf))

	// read last modified time
	_, err = f.file.ReadAt(int64Buf, lastModifiedOffset)
	if err != nil {
		log.Printf("Failed to read last modified time: %v", err)
		return false
	}
	f.LastModified = time.Unix(int64(binary.LittleEndian.Uint64(int64Buf)), 0)

	// read states
	stateBuf := make([]byte, statesLen)
	_, err = f.file.ReadAt(stateBuf, statesOffset)
	if err != nil {
		log.Printf("Failed to read states: %v", err)
	}
	f.Completed = stateBuf[0]&stateCompletedFlag == stateCompletedFlag

	// read trunks
	_, err = f.file.ReadAt(f.trunks, trunksOffset)
	if err != nil {
		log.Printf("Failed to read trunks: %v", err)
		return false
	}

	return true
}
