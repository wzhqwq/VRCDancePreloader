package rw_file

import (
	"context"
	"io"
)

type RSWithContext struct {
	file DeferredReader

	totalLength int64
	cursor      int64

	ctx context.Context
}

func NewRSWithContext(file DeferredReader, total int64, ctx context.Context) *RSWithContext {
	return &RSWithContext{
		file: file,

		totalLength: total,
		cursor:      0,

		ctx: ctx,
	}
}

func (rs *RSWithContext) Read(p []byte) (int, error) {
	select {
	case <-rs.ctx.Done():
		return 0, io.ErrClosedPipe
	default:
	}

	if rs.cursor >= rs.totalLength {
		return 0, io.EOF
	}

	//log.Println(rs.ID, "Request offset:", rs.cursor)
	err := rs.file.RequestRange(rs.cursor, int64(len(p)), rs.ctx)
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

func (rs *RSWithContext) Seek(offset int64, whence int) (int64, error) {
	select {
	case <-rs.ctx.Done():
		return 0, io.ErrClosedPipe
	default:
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
