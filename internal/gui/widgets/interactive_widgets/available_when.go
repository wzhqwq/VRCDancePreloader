package interactive_widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type AvailableWhen struct {
	LifeCycleWidget

	item fyne.CanvasObject

	// watch hands out one watcher per loop run. The widget needs exactly two
	// things from the state it follows — the value now and a wake-up for "it may
	// have changed" — and a utils.LevelWatcher provides both, so it never asks
	// the setting for a (recomputed) value and needs no interface of its own
	// (AGENTS.md §8 rule 1). A function rather than a stored watcher: the
	// renderer — and with it this widget instance — is built again whenever it
	// is recreated, while a watcher handed out here is closed when its loop ends.
	watch func() *utils.LevelWatcher[bool]

	UseDisable bool

	Reverse bool
}

func NewAvailableWhen(item fyne.CanvasObject, watch func() *utils.LevelWatcher[bool]) *AvailableWhen {
	a := &AvailableWhen{
		item: item,

		watch: watch,
	}
	a.AddLifeCycleFn(a.loop)
	a.ExtendBaseWidget(a)

	return a
}

func NewAvailableWhenNot(item fyne.CanvasObject, watch func() *utils.LevelWatcher[bool]) *AvailableWhen {
	a := &AvailableWhen{
		item: item,

		watch: watch,

		Reverse: true,
	}
	a.AddLifeCycleFn(a.loop)
	a.ExtendBaseWidget(a)

	return a
}

func (a *AvailableWhen) loop(stopCh <-chan struct{}) {
	watcher := a.watch()
	defer watcher.Close()

	a.applyAvailable(watcher.Current())

	for {
		select {
		case <-stopCh:
			return
		case <-watcher.Wakes():
			// A wake-up means "it may have changed": the value that caused it is
			// already stored in the watcher, so read that instead of re-deriving
			// the setting (Get on a derived setting recomputes).
			a.applyAvailable(watcher.Current())
		}
	}
}

func (a *AvailableWhen) applyAvailable(available bool) {
	if a.Reverse {
		available = !available
	}
	if a.UseDisable {
		if item, ok := a.item.(*widget.DisableableWidget); ok {
			if available {
				fyne.Do(item.Enable)
			} else {
				fyne.Do(item.Disable)
			}
		}
	} else {
		if available {
			fyne.Do(a.Show)
		} else {
			fyne.Do(a.Hide)
		}
	}
}

func (a *AvailableWhen) CreateRenderer() fyne.WidgetRenderer {
	r := &availableWhenRenderer{
		objects: []fyne.CanvasObject{a.item},
	}

	r.Created(a)

	return r
}

type availableWhenRenderer struct {
	BaseLifeCycleRenderer

	objects []fyne.CanvasObject
}

func (a *availableWhenRenderer) Layout(size fyne.Size) {
	a.objects[0].Move(fyne.NewPos(0, 0))
	a.objects[0].Resize(size)
}

func (a *availableWhenRenderer) MinSize() fyne.Size {
	return a.objects[0].MinSize()
}

func (a *availableWhenRenderer) Objects() []fyne.CanvasObject {
	return a.objects
}

func (a *availableWhenRenderer) Refresh() {
	a.objects[0].Refresh()
}
