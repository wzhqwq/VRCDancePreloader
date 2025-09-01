package utils

import "time"

type Lazy struct {
	lastUpdate time.Time
	changed    bool
	onUpdate   func()
}

func NewLazy(onUpdate func()) *Lazy {
	return &Lazy{
		lastUpdate: time.Unix(0, 0),
		onUpdate:   onUpdate,
	}
}

func (l *Lazy) Change() {
	if time.Now().Sub(l.lastUpdate) > time.Second {
		l.Update()
	} else {
		l.changed = true
	}
}

func (l *Lazy) WaitUpdate() <-chan time.Time {
	if l.changed {
		return time.After(max(time.Second-time.Now().Sub(l.lastUpdate), 0))
	}
	return nil
}

func (l *Lazy) Update() {
	l.changed = false
	l.lastUpdate = time.Now()
	l.onUpdate()
}
