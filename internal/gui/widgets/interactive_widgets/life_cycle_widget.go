package interactive_widgets

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2/widget"
)

type LifeCycleFn func(stopCh <-chan struct{})

type LifeCycleObject interface {
	AddLifeCycleFn(lifeCycleFn LifeCycleFn)
	LifeCycleFns() []LifeCycleFn
}

type LifeCycleWidget struct {
	widget.BaseWidget

	lifeCycleFns []LifeCycleFn
}

func (w *LifeCycleWidget) LifeCycleFns() []LifeCycleFn {
	return w.lifeCycleFns
}

func (w *LifeCycleWidget) AddLifeCycleFn(lifeCycleFn LifeCycleFn) {
	w.lifeCycleFns = append(w.lifeCycleFns, lifeCycleFn)
}

var _ LifeCycleObject = (*LifeCycleWidget)(nil)

type BaseLifeCycleRenderer struct {
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func (r *BaseLifeCycleRenderer) Created(wid LifeCycleObject) {
	stopCh := make(chan struct{})

	r.stopCh = stopCh

	for _, fn := range wid.LifeCycleFns() {
		r.wg.Go(func() {
			fn(stopCh)
		})
	}
}

func (r *BaseLifeCycleRenderer) Destroy() {
	if r.stopCh == nil {
		fmt.Println("Warning: BaseLifeCycleRenderer.Created not called before rendering")
		return
	}
	close(r.stopCh)
	r.wg.Wait()
}

type DisableableLifeCycleWidget struct {
	LifeCycleWidget

	disabled bool
}

func (w *DisableableLifeCycleWidget) Enable() {
	if !w.Disabled() {
		return // Enabled already
	}

	w.disabled = false

	w.Refresh()
}

func (w *DisableableLifeCycleWidget) Disable() {
	if w.Disabled() {
		return // Disabled already
	}

	w.disabled = true

	w.Refresh()
}

func (w *DisableableLifeCycleWidget) SetDisabledWithoutRefresh(disabled bool) {
	w.disabled = disabled
}

func (w *DisableableLifeCycleWidget) Disabled() bool {
	return w.disabled
}
