package third_parties

import "github.com/wzhqwq/VRCDancePreloader/internal/utils"

type IsAllowed struct {
	allowed bool

	em *utils.EventManager[bool]
}

func newIsAllowed(allowed bool, em *utils.EventManager[bool]) *IsAllowed {
	return &IsAllowed{allowed, em}
}

func (a *IsAllowed) SetAllowed(available bool) {
	if a.allowed != available {
		a.allowed = available
		a.em.NotifySubscribers(available)
	}
}
func (a *IsAllowed) NotifyIfAllowed() {
	if a.allowed {
		a.em.NotifySubscribers(a.allowed)
	}
}

type BothTrue struct {
	available, allowed bool

	satisfied bool

	em *utils.EventManager[bool]
}

func newBothTrue(available, allowed bool, em *utils.EventManager[bool]) *BothTrue {
	return &BothTrue{available, allowed, available && allowed, em}
}

func (b *BothTrue) BothSatisfied() bool {
	return b.satisfied
}

func (b *BothTrue) SetAvailable(available bool) {
	b.available = available

	b.satisfied = b.available && b.allowed
	b.NotifyIfSatisfied()
}

func (b *BothTrue) SetAllowed(allowed bool) {
	b.allowed = allowed

	satisfied := b.available && b.allowed
	if b.satisfied != satisfied {
		b.satisfied = satisfied
		b.em.NotifySubscribers(satisfied)
	}
}

func (b *BothTrue) NotifyIfSatisfied() {
	if b.satisfied {
		b.em.NotifySubscribers(b.satisfied)
	}
}
