package lists

import (
	"math"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/custom_fyne/containers/scroll"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type DataWithID interface {
	ID() string
}

type ListItem[T any] interface {
	RefreshWidget()
	Data() T
	Dirty() bool
	Removed() bool

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
			i.dirty = false
		})
	}
}
func (i *BaseListItem[T]) Data() T {
	return i.data
}
func (i *BaseListItem[T]) Dirty() bool {
	return i.dirty
}
func (i *BaseListItem[T]) Removed() bool {
	return i.removed
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

	Lazy    bool
	showTip bool

	scrolled     bool
	itemsChanged bool

	passivelyScrolled bool

	NoScroll  bool
	MinHeight float32

	scrollHeight  float32
	scrollOffset  float32
	contentHeight float32

	// public stubs
	SubscriberFn func() *utils.EventSubscriber[ListItemChange]
	RendererFn   func(item ListItem[T]) fyne.WidgetRenderer
	GetDataFn    func(id string) T
	ListDataFn   func(offset int) []T
	// private stubs
	scrolledFn func(offset float32)
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
		l.itemsChanged = true
		l.showTip = true
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
	l.itemsChanged = true

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
		l.itemsChanged = true
		l.showTip = true
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
	l.itemsChanged = true

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

	l.showTip = false
	l.itemsChanged = true
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
	l.itemsChanged = true

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
	if l.scrolledFn != nil {
		l.scrolledFn(offset)
	}
}

func (l *BaseList[T]) CreateRenderer() fyne.WidgetRenderer {
	bar := scroll.NewBar(scroll.Vertical, func(offset float32) {
		l.scrollOffset = offset
		l.passivelyScrolled = true
		l.Refresh()
		if l.scrolledFn != nil {
			l.scrolledFn(offset)
		}
	})

	text := canvas.NewText(i18n.T("message_lazy_list_changed"), theme.Color(theme.ColorNameForeground))
	text.TextSize = 14
	tip := container.NewStack(
		canvas.NewRectangle(theme.Color(custom_fyne.ColorNameOuterBackground)),
		container.NewCenter(container.NewPadded(text)),
	)

	r := &listRenderer[T]{
		l: l,

		container: &fyne.Container{Layout: &listLayout{}, Objects: l.items},
		bar:       bar,
		tip:       tip,

		stopCh: make(chan struct{}),
	}
	if l.SubscriberFn != nil {
		r.eventLoop(l.SubscriberFn())
	}

	return r
}

type listRenderer[T DataWithID] struct {
	l *BaseList[T]

	bar       *scroll.Bar
	container *fyne.Container
	tip       fyne.CanvasObject

	topPadding    float32
	bottomPadding float32

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
	listHeight := l.topPadding + l.container.MinSize().Height + l.bottomPadding
	if math.Abs(float64(listHeight-l.l.contentHeight)) > 1e-4 {
		l.l.contentHeight = listHeight
		l.bar.SetContentLength(listHeight)
	}

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

	l.container.Move(fyne.NewPos(0, l.topPadding-l.l.scrollOffset))
	l.container.Resize(fyne.NewSize(width, 0))
	if l.l.showTip {
		l.tip.Move(fyne.NewPos(0, 0))
		l.tip.Resize(fyne.NewSize(width, l.tip.MinSize().Height))
		l.tip.Show()
	} else {
		l.tip.Hide()
	}
}

func (l *listRenderer[T]) Destroy() {
	close(l.stopCh)
}

func (l *listRenderer[T]) Layout(size fyne.Size) {
	l.layout(size)
}

func (l *listRenderer[T]) MinSize() fyne.Size {
	listSize := l.container.MinSize()

	if !l.l.NoScroll {
		return fyne.NewSize(listSize.Width+l.bar.MinSize().Width, l.l.MinHeight)
	}

	return listSize
}

func (l *listRenderer[T]) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{l.container, l.bar, l.tip}
}

func (l *listRenderer[T]) Refresh() {
	if l.l.scrolled {
		l.bar.SetOffset(l.l.scrollOffset)
		l.layout(l.l.Size())
		l.l.scrolled = false
	}
	if l.l.passivelyScrolled {
		l.layout(l.l.Size())
		l.l.passivelyScrolled = false
	}

	if l.l.itemsChanged {
		if l.l.showTip {
			tipHeight := l.tip.MinSize().Height
			l.l.scrollOffset += tipHeight - l.topPadding
			l.topPadding = tipHeight
		} else {
			l.l.scrollOffset = max(0, l.l.scrollOffset-l.topPadding)
			l.topPadding = 0
		}

		l.container.Objects = l.l.items
		l.container.Layout.(*listLayout).itemsChanged = true
		l.layout(l.l.Size())
		l.l.itemsChanged = false
	}
}

type listLayout struct {
	itemsChanged bool

	height float32
}

func (*listLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	yOffset := float32(0)
	width := size.Width
	for _, obj := range objects {
		height := obj.MinSize().Height
		obj.Move(fyne.NewPos(0, yOffset))
		obj.Resize(fyne.NewSize(width, height))
		yOffset += height
	}
}

func (l *listLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minWidth := float32(0)
	if len(objects) > 0 {
		minWidth = objects[0].MinSize().Width
	}

	if l.itemsChanged {
		height := float32(0)
		for _, obj := range objects {
			height += obj.MinSize().Height
		}
		l.height = height
		l.itemsChanged = false
	}

	return fyne.NewSize(minWidth, l.height)
}
