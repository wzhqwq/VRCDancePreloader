package containers

import (
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

type DynamicListLayout struct {
	minWidth float32

	Order   []string
	ItemMap map[string]*DynamicListItem
}

func (dl *DynamicListLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	totalHeight := float32(0)
	for _, o := range objects {
		totalHeight += o.MinSize().Height + theme.Padding()
	}
	return fyne.NewSize(dl.minWidth+theme.Padding()*2, totalHeight+theme.Padding())
}
func (dl *DynamicListLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(size.Width, o.MinSize().Height))
	}
	accY := float32(0)
	for _, id := range dl.Order {
		if o, ok := dl.ItemMap[id]; ok {
			o.MoveY(accY)
			accY += o.MinSize().Height + theme.Padding()
		}
	}
}

type DynamicList struct {
	fyne.Container
	dlLayout *DynamicListLayout
}

func (dl *DynamicList) AddItem(item *DynamicListItem) {
	dl.Add(item)
	dl.dlLayout.ItemMap[item.ID] = item
	item.Move(fyne.NewPos(theme.Padding(), 0))
}
func (dl *DynamicList) RemoveItem(id string) {
	item := dl.dlLayout.ItemMap[id]
	if item == nil {
		return
	}
	dl.Remove(item)
	delete(dl.dlLayout.ItemMap, id)
}
func (dl *DynamicList) SetOrder(order []string) {
	dl.dlLayout.Order = order
	dl.Refresh()
}

func NewDynamicList(minWidth float32) *DynamicList {
	layout := &DynamicListLayout{
		minWidth: minWidth,

		Order:   []string{},
		ItemMap: map[string]*DynamicListItem{},
	}
	dl := &DynamicList{
		Container: fyne.Container{
			Objects: []fyne.CanvasObject{},
			Layout:  layout,
		},
		dlLayout: layout,
	}
	return dl
}

type DynamicListItem struct {
	fyne.Container
	ID               string
	runningAnimation *fyne.Animation
}

func (dli *DynamicListItem) MoveY(y float32) {
	if dli.Hidden {
		return
	}
	if math.Abs(float64(dli.Position().Y-y)) < 1e-3 {
		return
	}
	if dli.runningAnimation != nil {
		dli.runningAnimation.Stop()
	}
	dli.runningAnimation = canvas.NewPositionAnimation(
		dli.Position(),
		fyne.NewPos(dli.Position().X, y),
		500*time.Millisecond,
		dli.Move,
	)
	dli.runningAnimation.Start()
}

func NewDynamicListItem(ID string, objects ...fyne.CanvasObject) *DynamicListItem {
	return &DynamicListItem{
		Container: fyne.Container{
			Objects: objects,
			Layout:  layout.NewVBoxLayout(),
		},
		ID: ID,
	}
}
