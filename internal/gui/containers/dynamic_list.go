package containers

import (
	"fyne.io/fyne/v2/widget"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

type DynamicList struct {
	widget.BaseWidget

	minWidth float32
	order    []string
	itemMap  map[string]*DynamicListItem
}

func (dl *DynamicList) AddItem(item *DynamicListItem) {
	dl.itemMap[item.ID] = item
}
func (dl *DynamicList) RemoveItem(id string) {
	item := dl.itemMap[id]
	if item == nil {
		return
	}
	delete(dl.itemMap, id)
}
func (dl *DynamicList) SetOrder(order []string) {
	dl.order = order
	fyne.Do(func() {
		dl.Refresh()
	})
}
func (dl *DynamicList) CreateRenderer() fyne.WidgetRenderer {
	return &DynamicListRenderer{
		dl: dl,
	}
}

type DynamicListRenderer struct {
	dl *DynamicList
}

func (r *DynamicListRenderer) MinSize() fyne.Size {
	totalHeight := float32(0)
	for _, item := range r.dl.itemMap {
		totalHeight += item.MinSize().Height + theme.Padding()
	}
	return fyne.NewSize(r.dl.minWidth+theme.Padding()*2, totalHeight+theme.Padding())
}
func (r *DynamicListRenderer) Layout(size fyne.Size) {
	accY := theme.Padding()
	for _, id := range r.dl.order {
		if o, ok := r.dl.itemMap[id]; ok {
			o.Resize(fyne.NewSize(size.Width-theme.Padding()*2, o.MinSize().Height))
			if o.isRemoving {
				continue
			}
			if o.isNew {
				o.isNew = false
				o.goalY = accY
				o.Move(fyne.NewPos(theme.Padding(), accY))
			} else {
				o.SlideY(accY)
			}
			accY += o.MinSize().Height + theme.Padding()
		}
	}
}
func (r *DynamicListRenderer) Refresh() {
	for _, item := range r.dl.itemMap {
		item.Refresh()
	}
}
func (r *DynamicListRenderer) Objects() []fyne.CanvasObject {
	var objs []fyne.CanvasObject
	for _, item := range r.dl.itemMap {
		objs = append(objs, item)
	}
	return objs
}
func (r *DynamicListRenderer) Destroy() {
	// TODO
}

func NewDynamicList(minWidth float32) *DynamicList {
	dl := &DynamicList{
		minWidth: minWidth,
		order:    []string{},
		itemMap:  map[string]*DynamicListItem{},
	}
	dl.ExtendBaseWidget(dl)
	return dl
}

type DynamicListItem struct {
	widget.BaseWidget
	ID     string
	object fyne.CanvasObject
	dl     *DynamicList

	isNew      bool
	isRemoving bool

	goalY float32

	runningAnimation *fyne.Animation
}

func (dli *DynamicListItem) CreateRenderer() fyne.WidgetRenderer {
	return &DynamicListItemRenderer{
		dli: dli,
	}
}

func (dli *DynamicListItem) SlideY(y float32) {
	if dli.Hidden {
		return
	}
	if math.Abs(float64(dli.goalY-y)) < 1e-3 {
		return
	}
	dli.goalY = y
	if dli.runningAnimation != nil {
		dli.runningAnimation.Stop()
	}
	dli.runningAnimation = canvas.NewPositionAnimation(
		dli.Position(),
		fyne.NewPos(dli.Position().X, y),
		300*time.Millisecond,
		dli.Move,
	)
	dli.runningAnimation.Start()
}

func (dli *DynamicListItem) MarkRemoving() {
	dli.isRemoving = true
}

func (dli *DynamicListItem) NotifyUpdateMinSize() {
	dli.dl.Refresh()
}

func NewDynamicListItem(ID string, dl *DynamicList, object fyne.CanvasObject) *DynamicListItem {
	dli := &DynamicListItem{
		ID:     ID,
		object: object,
		dl:     dl,

		isNew:      true,
		isRemoving: false,
	}
	dli.ExtendBaseWidget(dli)
	return dli
}

type DynamicListItemRenderer struct {
	dli *DynamicListItem
}

func (r *DynamicListItemRenderer) MinSize() fyne.Size {
	return r.dli.object.MinSize()
}
func (r *DynamicListItemRenderer) Layout(size fyne.Size) {
	r.dli.object.Resize(size)
}
func (r *DynamicListItemRenderer) Refresh() {
	r.dli.object.Refresh()
}
func (r *DynamicListItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dli.object}
}
func (r *DynamicListItemRenderer) Destroy() {
	// TODO
}
