package task

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
	}

	c.window[c.p].size += size
	c.window[c.p].passed = time.Since(c.lastTime)

	if c.window[c.p].passed > time.Millisecond {
		c.lastTime = time.Now()
		c.p = (c.p + 1) % MaxWindowSize
		c.window[c.p] = etaSlice{}
	}

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

func (c *etaCalculator) QueryEta() (time.Time, bool) {
	speed := c.QuerySpeed()
	if c.goal == 0 || speed == 0 {
		return time.Time{}, false
	}

	remaining := c.goal - c.achieved
	if remaining <= 0 {
		return time.Now(), true
	}

	remainTime := float64(remaining) / speed

	return time.Now().Add(time.Duration(remainTime * float64(time.Second))), true
}

func (c *etaCalculator) QueryRemainTime() time.Duration {
	speed := c.QuerySpeed()
	if c.goal == 0 || speed == 0 {
		return -1
	}

	remaining := c.goal - c.achieved
	if remaining <= 0 {
		return 0
	}

	remainTime := float64(remaining) / speed

	return time.Duration(remainTime * float64(time.Second))
}

func (c *etaCalculator) Passed() time.Duration {
	return time.Since(c.startTime)
}
