package utils

import (
	"context"
	"errors"
	"io"
	"time"
)

type BodyWithContext struct {
	ctx  context.Context
	body io.ReadCloser
}

func NewBodyWithContext(ctx context.Context, body io.ReadCloser) *BodyWithContext {
	return &BodyWithContext{
		ctx:  ctx,
		body: body,
	}
}

func (c *BodyWithContext) Read(p []byte) (int, error) {
	n, err := c.body.Read(p)

	if err != nil {
		if cause := context.Cause(c.ctx); errors.Is(err, context.Canceled) && cause != nil {
			return n, cause
		}
	}

	return n, err
}

func (c *BodyWithContext) Close() error {
	return c.body.Close()
}

type BodyWithBandwidth struct {
	body  io.ReadCloser
	delay time.Duration

	lastTimeTrunkFull time.Time
	trunkSize         int

	closeCh chan struct{}
}

const bytesInMBit = 1024 * 1024 / 8

type PacingReader struct {
	r io.ReadSeeker

	bytesPerTick int
	tick         time.Duration

	allowance int
	lastTick  time.Time

	timer *time.Timer
}

func NewPacingReader(r io.ReadSeeker, mbps int64) *PacingReader {
	bytesPerSec := mbps * bytesInMBit
	tick := 10 * time.Millisecond
	return &PacingReader{
		r:            r,
		bytesPerTick: int(bytesPerSec * int64(tick) / int64(time.Second)),
		tick:         tick,
		lastTick:     time.Now(),
		timer:        time.NewTimer(0),
	}
}

func (p *PacingReader) Read(b []byte) (int, error) {
	for {
		now := time.Now()

		elapsed := now.Sub(p.lastTick)
		if elapsed >= p.tick {
			ticks := int(elapsed / p.tick)
			p.allowance += ticks * p.bytesPerTick
			if p.allowance > p.bytesPerTick {
				p.allowance = p.bytesPerTick
			}
			p.lastTick = p.lastTick.Add(time.Duration(ticks) * p.tick)
		}

		if p.allowance > 0 {
			n := len(b)
			if n > p.allowance {
				n = p.allowance
			}

			readN, err := p.r.Read(b[:n])
			p.allowance -= readN

			return readN, err
		}

		wait := p.tick - elapsed
		if wait < 0 {
			wait = p.tick
		}

		if !p.timer.Stop() {
			select {
			case <-p.timer.C:
			default:
			}
		}
		p.timer.Reset(wait)
		<-p.timer.C
	}
}

func (p *PacingReader) Seek(offset int64, whence int) (int64, error) {
	return p.r.Seek(offset, whence)
}
