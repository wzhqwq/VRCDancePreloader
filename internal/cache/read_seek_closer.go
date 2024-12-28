package cache

import (
	"io"
)

type rsc struct {
	file *ReadableWritingFile

	totalLength     int64
	availableLength int64

	cursor int64
	Closed bool

	availableLengthCh chan int64
	closeCh           chan struct{}
}

func newRsc(file *ReadableWritingFile, total int64) *rsc {
	return &rsc{
		file: file,

		totalLength:     total,
		availableLength: file.size,

		cursor: 0,
		Closed: false,

		availableLengthCh: make(chan int64, 10),
		closeCh:           make(chan struct{}),
	}
}

func (rs *rsc) Read(p []byte) (n int, err error) {
	if rs.Closed {
		return 0, io.ErrClosedPipe
	}
	if rs.cursor >= rs.totalLength {
		return 0, io.EOF
	}
	for {
		if rs.availableLength < 0 {
			return 0, io.ErrClosedPipe
		}
		if rs.cursor >= rs.availableLength {
			select {
			case rs.availableLength = <-rs.availableLengthCh:
			case <-rs.closeCh:
				return 0, io.ErrClosedPipe
			}
		} else {
			break
		}
	}
	movement := min(len(p), int(rs.availableLength-rs.cursor))

	_, err = rs.file.ReadAt(p[:movement], rs.cursor)
	if err != nil {
		return 0, err
	}

	rs.cursor += int64(movement)
	//log.Printf("Read cursor %d/%d/%d", rs.cursor, rs.availableLength, rs.totalLength)
	return movement, nil
}

func (rs *rsc) Seek(offset int64, whence int) (int64, error) {
	if rs.Closed {
		return 0, io.ErrClosedPipe
	}
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

func (rs *rsc) Close() error {
	close(rs.closeCh)
	rs.Closed = true
	return nil
}
