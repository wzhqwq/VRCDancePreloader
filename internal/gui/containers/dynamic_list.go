package containers

import (
	"fyne.io/fyne/v2/widget"
	"math"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type DynamicList struct {
	widget.BaseWidget

	Gap      float32
	Padding  float32
	MinWidth float32

	order   []int64
	itemMap map[int64]*DynamicListItem

	renderedItemsChanged bool

	runningAnimation *fyne.Animation
}

func (dl *DynamicList) AddItem(item *DynamicListItem, transition bool) {
	if i, ok := dl.itemMap[item.ID]; ok {
		item.copyStateFrom(i)
	}
	dl.itemMap[item.ID] = item
	if transition && dl.Size().Width > 0 {
		item.enableEnteringTransition = true
	}
}
func (dl *DynamicList) RemoveItem(id int64, transition bool) {
	item := dl.itemMap[id]
	if item == nil {
		return
	}
	item.enableExitingTransition = transition
	item.exit()
	dl.renderedItemsChanged = true
	fyne.Do(func() {
		dl.Refresh()
	})
}
func (dl *DynamicList) SetOrder(order []int64) {
	dl.order = order
	dl.renderedItemsChanged = true
	dl.Refresh()
}
func (dl *DynamicList) CreateRenderer() fyne.WidgetRenderer {
	return &DynamicListRenderer{
		dl: dl,
	}
}
func (dl *DynamicList) removeFromMap(id int64) {
	delete(dl.itemMap, id)
}

type DynamicListRenderer struct {
	dl      *DynamicList
	objects []fyne.CanvasObject

	minHeight float32
}

func (r *DynamicListRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.dl.MinWidth+r.dl.Padding*2, r.minHeight)
}
func (r *DynamicListRenderer) Layout(size fyne.Size) {
	for _, o := range r.objects {
		if item, ok := o.(*DynamicListItem); ok {
			item.Resize(fyne.NewSize(size.Width-r.dl.Padding*2, o.MinSize().Height))
		}
	}
}
func (r *DynamicListRenderer) layoutPosition() {
	accY := r.dl.Padding
	for _, id := range r.dl.order {
		if item, ok := r.dl.itemMap[id]; ok {
			if !item.IsExiting() {
				//fmt.Printf("%s %f\n", item.ID, accY)
				item.setY(accY)
				accY += item.MinSize().Height + r.dl.Gap
			}
		}
	}

	animationNeeded := false
	for _, o := range r.objects {
		if item, ok := o.(*DynamicListItem); ok {
			if item.state.Load() != int32(Placed) {
				animationNeeded = true
			}
		}
	}
	if animationNeeded {
		if r.dl.runningAnimation != nil {
			r.dl.runningAnimation.Stop()
		}
		r.dl.runningAnimation = &fyne.Animation{
			Duration: time.Millisecond * 300,
			Tick: func(d float32) {
				for _, o := range r.objects {
					if item, ok := o.(*DynamicListItem); ok {
						item.tick(d)
					}
				}
			},
		}
		r.dl.runningAnimation.Start()
	}
}
func (r *DynamicListRenderer) updateObjects() {
	r.objects = make([]fyne.CanvasObject, 0, len(r.dl.itemMap))
	r.minHeight = r.dl.Padding*2 - r.dl.Gap
	for _, item := range r.dl.itemMap {
		if item.IsExiting() {
			r.objects = append(r.objects, item)
		}
	}
	for _, item := range r.dl.itemMap {
		if item.IsInactive() {
			r.dl.removeFromMap(item.ID)
		} else {
			if !item.IsExiting() {
				r.objects = append(r.objects, item)
			}
			r.minHeight += item.MinSize().Height + r.dl.Gap
		}
	}
}
func (r *DynamicListRenderer) Refresh() {
	if r.dl.renderedItemsChanged {
		r.dl.renderedItemsChanged = false
		r.updateObjects()
		r.layoutPosition()
	}
	for _, item := range r.dl.itemMap {
		item.Refresh()
	}
}
func (r *DynamicListRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
func (r *DynamicListRenderer) Destroy() {
	// TODO
}

func NewDynamicList(minWidth float32) *DynamicList {
	dl := &DynamicList{
		Gap:     theme.Padding(),
		Padding: theme.Padding(),

		MinWidth: minWidth,
		order:    make([]int64, 0),
		itemMap:  map[int64]*DynamicListItem{},
	}
	dl.ExtendBaseWidget(dl)
	return dl
}

type ListItemType int32

const (
	// Initial - Just created and not positioned
	Initial ListItemType = iota
	// Placed - Have been placed to one exact position and animations can start now
	Placed
	// OnEnterTransition - It's running entering animation
	OnEnterTransition
	// Sliding - It's running sliding animation due to layout
	Sliding
	// OnExitTransition - It's running exiting animation, can be reverted
	OnExitTransition
	// Inactive - Do not render it anymore
	Inactive
)

type DynamicListItem struct {
	widget.BaseWidget
	ID     int64
	object fyne.CanvasObject
	dl     *DynamicList

	state atomic.Int32

	enableEnteringTransition bool
	enableExitingTransition  bool

	startPosition fyne.Position
	goalPosition  fyne.Position
}

func (i *DynamicListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.object)
}

func (i *DynamicListItem) setY(y float32) {
	if i.state.CompareAndSwap(int32(Initial), int32(Placed)) {
		x := i.dl.Padding
		if i.enableEnteringTransition {
			x = i.dl.Size().Width + 10
		}
		//fmt.Printf("place %f, %f\n", x, y)
		i.Move(fyne.NewPos(x, y))
		i.enter()
	} else if math.Abs(float64(i.Position().Y-y)) > 0.1 {
		i.startPosition = i.Position()
		if i.state.CompareAndSwap(int32(Placed), int32(Sliding)) {
			i.goalPosition = fyne.NewPos(i.startPosition.X, y)
			//fmt.Printf("slide: %f, %f\n", i.goalPosition.X, i.goalPosition.Y)
		} else {
			i.goalPosition = fyne.NewPos(i.goalPosition.X, y)
			//fmt.Printf("slide (continued): %f, %f\n", i.goalPosition.X, i.goalPosition.Y)
		}
	}
}

func (i *DynamicListItem) enter() {
	if i.enableEnteringTransition && i.state.CompareAndSwap(int32(Placed), int32(OnEnterTransition)) {
		i.startPosition = i.Position()
		i.goalPosition = fyne.NewPos(i.dl.Padding, i.startPosition.Y)
		//fmt.Printf("enter: %f, %f\n", i.goalPosition.X, i.goalPosition.Y)
	}
}

func (i *DynamicListItem) exit() {
	if i.enableExitingTransition {
		goalX := -i.dl.Size().Width - 10
		old := i.state.Swap(int32(OnExitTransition))
		i.startPosition = i.Position()

		switch ListItemType(old) {
		case Placed:
			// start animation, change x only
			i.goalPosition = fyne.NewPos(goalX, i.startPosition.Y)
			break
			//fmt.Printf("exit: %f, %f\n", i.goalPosition.X, i.goalPosition.Y)
		case OnEnterTransition:
		case Sliding:
			// continue animation
			i.goalPosition = fyne.NewPos(goalX, i.goalPosition.Y)
			break
			//fmt.Printf("exit (continued): %f, %f\n", i.goalPosition.X, i.goalPosition.Y)
		case Initial:
		case Inactive:
			// swap back
			i.state.Swap(int32(Inactive))
			break
		default:
			break
		}
	} else {
		i.state.Swap(int32(Inactive))
	}
}

func (i *DynamicListItem) copyStateFrom(another *DynamicListItem) {
	i.state.Swap(another.state.Load())
	i.startPosition = another.startPosition
	i.goalPosition = another.goalPosition
	i.Move(another.Position())
}

func (i *DynamicListItem) tick(progress float32) {
	if i.IsAnimating() {
		i.Move(fyne.NewPos(
			i.startPosition.X*(1-progress)+i.goalPosition.X*progress,
			i.startPosition.Y*(1-progress)+i.goalPosition.Y*progress,
		))
		if progress+1e-4 > 1.0 {
			if i.state.CompareAndSwap(int32(OnExitTransition), int32(Inactive)) {
				i.NotifyUpdateMinSize()
			} else {
				i.state.Swap(int32(Placed))
			}
		}
	}
}

func (i *DynamicListItem) IsInactive() bool {
	return i.state.Load() == int32(Inactive)
}

func (i *DynamicListItem) IsExiting() bool {
	return i.state.Load() == int32(OnExitTransition)
}

func (i *DynamicListItem) IsAnimating() bool {
	s := ListItemType(i.state.Load())
	return s == Sliding || s == OnEnterTransition || s == OnExitTransition
}

func (i *DynamicListItem) NotifyUpdateMinSize() {
	go func() {
		fyne.Do(func() {
			i.dl.renderedItemsChanged = true
			i.dl.Refresh()
		})
	}()
}

func NewDynamicListItem(ID int64, dl *DynamicList, object fyne.CanvasObject) *DynamicListItem {
	dli := &DynamicListItem{
		ID:     ID,
		object: object,
		dl:     dl,
	}
	dli.ExtendBaseWidget(dli)
	return dli
}
