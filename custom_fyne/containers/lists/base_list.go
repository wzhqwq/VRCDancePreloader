package lists

import (
	"math"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/custom_fyne/containers/scroll"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type DataWithID interface {
	ID() string
}

type ListItem[T any] interface {
	RefreshWidget()
	Data() T
	Dirty() bool

	updateData(T)
	setActive(bool)
	setRemoved()
}

type ReusableList[T any] interface {
	ConfigureAllStubs(
		SubscriberFn func() *utils.EventSubscriber[ListItemChange],
		RendererFn func(item ListItem[T]) fyne.WidgetRenderer,
		GetDataFn func(id string) T,
		ListDataFn func(offset int) []T,
	)
	RefreshItems()
	Data(index int) T

	itemRenderer(item ListItem[T]) fyne.WidgetRenderer

	isActive(index int) bool
}

type ListItemChange struct {
	Op byte
	ID string
}

type BaseListItem[T any] struct {
	widget.BaseWidget
	ListItem[T]

	list ReusableList[T]

	data T

	dirty   bool
	active  bool
	removed bool
}

var _ ListItem[any] = (*BaseListItem[any])(nil)

func (i *BaseListItem[T]) ExtendBaseListItem(list ReusableList[T], index int, super fyne.Widget) {
	i.list = list
	i.data = list.Data(index)
	i.active = list.isActive(index)

	i.ExtendBaseWidget(super)
}

func (i *BaseListItem[T]) CreateRenderer() fyne.WidgetRenderer {
	return i.list.itemRenderer(i)
}

func (i *BaseListItem[T]) RefreshWidget() {
	if i.active {
		fyne.Do(func() {
			i.Refresh()
		})
		i.dirty = false
	}
}
func (i *BaseListItem[T]) Data() T {
	return i.data
}
func (i *BaseListItem[T]) Dirty() bool {
	return i.dirty
}

func (i *BaseListItem[T]) updateData(data T) {
	i.data = data
	i.dirty = true
	i.RefreshWidget()
}
func (i *BaseListItem[T]) setActive(active bool) {
	i.active = active
	if i.dirty && active {
		i.RefreshWidget()
	}
}
func (i *BaseListItem[T]) setRemoved() {
	i.removed = true
	i.RefreshWidget()
}

type BaseList[T DataWithID] struct {
	widget.BaseWidget
	ReusableList[T]
	fyne.Scrollable

	Lazy bool

	changed  bool
	scrolled bool

	passivelyScrolled bool

	NoScroll  bool
	MinHeight float32

	scrollHeight  float32
	scrollOffset  float32
	contentHeight float32

	// stub
	SubscriberFn func() *utils.EventSubscriber[ListItemChange]
	RendererFn   func(item ListItem[T]) fyne.WidgetRenderer
	GetDataFn    func(id string) T
	ListDataFn   func(offset int) []T
	// for override
	newItemFn func(index int) ListItem[T]

	data  []T
	items []fyne.CanvasObject

	itemMap map[string]ListItem[T]
}

var _ ReusableList[DataWithID] = (*BaseList[DataWithID])(nil)
var _ fyne.Scrollable = (*BaseList[DataWithID])(nil)

func NewBaseList[T DataWithID]() *BaseList[T] {
	l := &BaseList[T]{}
	l.ExtendBaseList(l)
	return l
}

func (l *BaseList[T]) ExtendBaseList(super fyne.Widget) {
	l.newItemFn = func(index int) ListItem[T] {
		item := &BaseListItem[T]{}
		item.ExtendBaseListItem(l, index, item)
		return item
	}

	l.ExtendBaseWidget(super)
}

func (l *BaseList[T]) ConfigureAllStubs(
	SubscriberFn func() *utils.EventSubscriber[ListItemChange],
	RendererFn func(item ListItem[T]) fyne.WidgetRenderer,
	GetDataFn func(id string) T,
	ListDataFn func(offset int) []T,
) {
	l.SubscriberFn = SubscriberFn
	l.RendererFn = RendererFn
	l.GetDataFn = GetDataFn
	l.ListDataFn = ListDataFn
}

func (l *BaseList[T]) itemRenderer(item ListItem[T]) fyne.WidgetRenderer {
	return l.RendererFn(item)
}

func (l *BaseList[T]) isActive(index int) bool {
	return true
}
func (l *BaseList[T]) Data(index int) T {
	return l.data[index]
}

func (l *BaseList[T]) newItem(index int) (ListItem[T], fyne.CanvasObject) {
	item := l.newItemFn(index)
	if obj, ok := item.(fyne.CanvasObject); ok {
		return item, obj
	}
	panic("item must be a canvas object")
}

func (l *BaseList[T]) addItem(id string) {
	if l.Lazy {
		l.changed = true
		fyne.Do(func() {
			l.Refresh()
		})
		return
	}

	if l.GetDataFn == nil {
		return
	}
	l.data = append(l.data, l.GetDataFn(id))

	item, obj := l.newItem(len(l.items))
	l.items = append(l.items, obj)
	l.itemMap[id] = item
	fyne.Do(func() {
		l.Refresh()
	})
}

func (l *BaseList[T]) removeItem(id string) {
	if l.Lazy {
		if item, ok := l.itemMap[id]; ok {
			item.setRemoved()
			delete(l.itemMap, id)
		}
		l.changed = true
		fyne.Do(func() {
			l.Refresh()
		})
		return
	}

	if item, ok := l.itemMap[id]; ok {
		obj := item.(fyne.CanvasObject)
		index := lo.IndexOf(l.items, obj)
		if index != -1 {
			l.items = slices.Delete(l.items, index, index+1)
			l.data = slices.Delete(l.data, index, index+1)
		}
		delete(l.itemMap, id)
	}
	fyne.Do(func() {
		l.Refresh()
	})
}

func (l *BaseList[T]) updateItem(id string) {
	if l.GetDataFn == nil {
		return
	}
	if item, ok := l.itemMap[id]; ok {
		item.updateData(l.GetDataFn(id))
	}
}

func (l *BaseList[T]) RefreshItems() {
	if l.ListDataFn == nil {
		return
	}

	l.data = l.ListDataFn(0)
	l.itemMap = make(map[string]ListItem[T])
	l.items = lo.Map(l.data, func(data T, index int) fyne.CanvasObject {
		item, obj := l.newItem(index)
		l.itemMap[data.ID()] = item
		return obj
	})

	l.changed = false
	fyne.Do(func() {
		l.Refresh()
	})
}
func (l *BaseList[T]) appendItems() {
	if l.ListDataFn == nil {
		return
	}

	newData := l.ListDataFn(len(l.data))
	if len(newData) == 0 {
		return
	}

	l.data = append(l.data, newData...)
	for _, data := range newData {
		item, obj := l.newItem(len(l.items))
		l.items = append(l.items, obj)
		l.itemMap[data.ID()] = item
	}

	fyne.Do(func() {
		l.Refresh()
	})
}

func (l *BaseList[T]) Scrolled(e *fyne.ScrollEvent) {
	offset := max(0, min(l.scrollHeight, l.scrollOffset-e.Scrolled.DY))

	if l.NoScroll || e.Scrolled.IsZero() || math.Abs(float64(offset-l.scrollOffset)) < 0.5 {
		return
	}

	l.scrollOffset = offset
	l.scrolled = true
	l.Refresh()
}

func (l *BaseList[T]) CreateRenderer() fyne.WidgetRenderer {
	r := &listRenderer[T]{
		l: l,

		bar: scroll.NewBar(scroll.Vertical, func(offset float32) {
			l.scrollOffset = offset
			l.passivelyScrolled = true
			l.Refresh()
		}),

		stopCh: make(chan struct{}),
	}
	if l.SubscriberFn != nil {
		r.eventLoop(l.SubscriberFn())
	}

	return r
}

type listRenderer[T DataWithID] struct {
	l *BaseList[T]

	bar *scroll.Bar

	stopCh chan struct{}
}

func (l *listRenderer[T]) eventLoop(sub *utils.EventSubscriber[ListItemChange]) {
	go func() {
		defer sub.Close()
		for {
			select {
			case e := <-sub.Channel:
				switch e.Op {
				case '+':
					l.l.addItem(e.ID)
				case '-':
					l.l.removeItem(e.ID)
				case '*':
					l.l.updateItem(e.ID)
				}
			case <-l.stopCh:
				return
			}
		}
	}()
}

func (l *listRenderer[T]) layout(size fyne.Size) {
	l.l.scrollHeight = max(0, l.l.contentHeight-size.Height)
	width := size.Width

	if !l.l.NoScroll && l.l.scrollHeight > 0.5 {
		barWidth := l.bar.MinSize().Width
		width -= barWidth
		l.bar.Show()
		l.bar.Move(fyne.NewPos(width, 0))
		l.bar.Resize(fyne.NewSize(barWidth, size.Height))
	} else {
		l.bar.Hide()
	}

	yOffset := -l.l.scrollOffset
	objs := l.l.items
	for _, obj := range objs {
		height := obj.MinSize().Height
		obj.Move(fyne.NewPos(0, yOffset))
		obj.Resize(fyne.NewSize(width, height))
		yOffset += height
	}
}

func (l *listRenderer[T]) Destroy() {
	close(l.stopCh)
}

func (l *listRenderer[T]) Layout(size fyne.Size) {
	l.layout(size)
}

func (l *listRenderer[T]) MinSize() fyne.Size {
	minWidth := float32(0)
	objs := l.l.items
	if len(objs) > 0 {
		minWidth = objs[0].MinSize().Width
	}

	if !l.l.NoScroll {
		return fyne.NewSize(minWidth-l.bar.MinSize().Width, l.l.MinHeight)
	}

	return fyne.NewSize(minWidth, l.l.contentHeight)
}

func (l *listRenderer[T]) Objects() []fyne.CanvasObject {
	return append(l.l.items, l.bar)
}

func (l *listRenderer[T]) Refresh() {
	if l.l.Lazy && l.l.changed {
		// TODO: render tip
	}
	if l.l.scrolled {
		l.bar.SetOffset(l.l.scrollOffset)
		l.layout(l.l.Size())
		l.l.scrolled = false
	}
	if l.l.passivelyScrolled {
		l.layout(l.l.Size())
		l.l.passivelyScrolled = false
	}

	totalHeight := float32(0)

	objs := l.l.items
	for _, obj := range objs {
		obj.(ListItem[T]).RefreshWidget()
		totalHeight += obj.MinSize().Height
	}

	if math.Abs(float64(totalHeight-l.l.contentHeight)) > 1e-4 {
		l.l.contentHeight = totalHeight
		l.bar.SetContentLength(totalHeight)
		l.layout(l.l.Size())
	}
}
