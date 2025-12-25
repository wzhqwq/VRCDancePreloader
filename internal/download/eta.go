package download

import "time"

type etaSlice struct {
	size   int64
	passed time.Duration
}

const MaxWindowSize = 10

type etaCalculator struct {
	window []etaSlice

	p        int
	goal     int64
	achieved int64

	lastTime  time.Time
	startTime time.Time
}

func newEtaCalculator(goal int64) *etaCalculator {
	return &etaCalculator{
		window:    make([]etaSlice, MaxWindowSize),
		p:         0,
		goal:      goal,
		startTime: time.Now(),
	}
}

func (c *etaCalculator) Add(size int64) {
	if c.lastTime.IsZero() {
		c.lastTime = time.Now()
		return
	}
	c.window[c.p] = etaSlice{size, time.Now().Sub(c.lastTime)}
	c.p = (c.p + 1) % MaxWindowSize
	c.achieved += size
}

func (c *etaCalculator) QuerySpeed() float64 {
	var sizeSum int64
	var timeSum time.Duration

	for _, s := range c.window {
		sizeSum += s.size
		timeSum += s.passed
	}

	if sizeSum == 0 || timeSum == 0 {
		return 0
	}

	return float64(sizeSum) / timeSum.Seconds()
}

func (c *etaCalculator) QueryEta() (time.Duration, bool) {
	speed := c.QuerySpeed()
	if c.goal == 0 || speed == 0 {
		return 0, false
	}

	remaining := c.goal - c.achieved
	if remaining <= 0 {
		return 0, true
	}

	eta := float64(remaining) / speed

	return time.Duration(eta * float64(time.Second)), true
}

func (c *etaCalculator) Passed() time.Duration {
	return time.Since(c.startTime)
}
