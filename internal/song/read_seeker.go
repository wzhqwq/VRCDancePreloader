package song

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"io"
	"log"
)

func (ps *PreloadedSong) GetSongRSSync() (io.ReadSeekCloser, error) {
	err := ps.sm.WaitForCompleteSong()
	if err != nil {
		return nil, err
	}
	return cache.OpenCache(ps.GetId()), nil
}

func (ps *PreloadedSong) GetSongRSAsync() (io.ReadSeekCloser, error) {
	closeCh := make(chan struct{})
	total, availableCh, err := ps.sm.SubscribePartialDownload(closeCh)
	if err != nil {
		return nil, err
	}
	rs := &readSeekerAsync{
		totalLength:     total,
		availableLength: <-availableCh,

		availableLengthCh: availableCh,
		closeCh:           closeCh,
	}
	return rs, nil
}

type readSeekerAsync struct {
	totalLength     int64
	availableLength int64

	cursor int64

	availableLengthCh chan int64
	closeCh           chan struct{}
}

func (rs *readSeekerAsync) Read(p []byte) (n int, err error) {
	if rs.cursor >= rs.totalLength {
		return 0, io.EOF
	}
	for {
		if rs.availableLength < 0 {
			return 0, io.EOF
		}
		if rs.cursor >= rs.availableLength {
			select {
			case rs.availableLength = <-rs.availableLengthCh:
			case <-rs.closeCh:
				return 0, io.EOF
			}

		} else {
			break
		}
	}
	movement := min(len(p), int(rs.availableLength-rs.cursor))
	rs.cursor += int64(movement)
	log.Printf("Read cursor %d/%d/%d", rs.cursor, rs.availableLength, rs.totalLength)
	return movement, nil
}

func (rs *readSeekerAsync) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		rs.cursor = offset
	case io.SeekCurrent:
		rs.cursor += offset
	case io.SeekEnd:
		rs.cursor = offset + rs.totalLength
	}
	if rs.cursor < 0 {
		rs.cursor = 0
		return 0, io.EOF
	}
	if rs.cursor > rs.totalLength {
		rs.cursor = 0
		return 0, io.EOF
	}
	return rs.cursor, nil
}

func (rs *readSeekerAsync) Close() error {
	close(rs.closeCh)
	return nil
}
