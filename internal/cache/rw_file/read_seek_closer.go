package rw_file

import (
	"io"
)

var rscIncrement = 0

type Rsc struct {
	file DeferredReader

	ID int

	totalLength int64
	cursor      int64

	Closed bool

	closeCh chan struct{}
}

func NewRsc(file DeferredReader, total int64) *Rsc {
	rscIncrement++
	return &Rsc{
		file: file,

		ID: rscIncrement,

		totalLength: total,

		cursor: 0,
		Closed: false,

		closeCh: make(chan struct{}),
	}
}

func (rs *Rsc) Read(p []byte) (int, error) {
	if rs.Closed {
		return 0, io.ErrClosedPipe
	}
	if rs.cursor >= rs.totalLength {
		return 0, io.EOF
	}

	//log.Println(rs.ID, "Request offset:", rs.cursor)
	err := rs.file.RequestRange(rs.cursor, int64(len(p)), rs.closeCh)
	if err != nil {
		return 0, err
	}

	n, err := rs.file.ReadAt(p, rs.cursor)
	if err != nil {
		return 0, err
	}
	rs.cursor += int64(n)
	return n, nil
}

func (rs *Rsc) Seek(offset int64, whence int) (int64, error) {
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
	//log.Println(rs.ID, "Seek offset:", rs.cursor)
	return rs.cursor, nil
}

func (rs *Rsc) Close() error {
	//log.Println(rs.ID, "rsc closed, cursor:", rs.cursor)
	close(rs.closeCh)
	rs.Closed = true
	return nil
}
