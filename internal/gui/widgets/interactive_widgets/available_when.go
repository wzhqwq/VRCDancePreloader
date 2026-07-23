package interactive_widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type AvailableWhen struct {
	LifeCycleWidget

	item fyne.CanvasObject

	getAndSub GetAndSubBool

	UseDisable bool

	Reverse bool
}

type GetAndSubBool interface {
	Get() bool
	Subscribe() *utils.EventSubscriber[bool]
}

func NewAvailableWhen(item fyne.CanvasObject, getAndSub GetAndSubBool) *AvailableWhen {
	a := &AvailableWhen{
		item: item,

		getAndSub: getAndSub,
	}
	a.AddLifeCycleFn(a.loop)
	a.ExtendBaseWidget(a)

	return a
}

func NewAvailableWhenNot(item fyne.CanvasObject, getAndSub GetAndSubBool) *AvailableWhen {
	a := &AvailableWhen{
		item: item,

		getAndSub: getAndSub,

		Reverse: true,
	}
	a.AddLifeCycleFn(a.loop)
	a.ExtendBaseWidget(a)

	return a
}

func (a *AvailableWhen) loop(stopCh <-chan struct{}) {
	ch := a.getAndSub.Subscribe()
	defer ch.Close()

	a.applyAvailable(a.getAndSub.Get())

	for {
		select {
		case <-stopCh:
			return
		case available := <-ch.Channel:
			a.applyAvailable(available)
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
