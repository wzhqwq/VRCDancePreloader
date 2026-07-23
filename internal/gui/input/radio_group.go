package input

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

// Copied from fyne widget and simplified

type NamedOption[T AcceptedValue] struct {
	Value T
	Label string
}

type RadioGroup[T AcceptedValue] struct {
	interactive_widgets.DisableableLifeCycleWidget

	Horizontal bool

	Options []NamedOption[T]
	Label   string

	LastSelected  int
	SelectedIndex int

	LastDisabled bool

	setting interactive.StatefulSetting[T]
}

var _ fyne.Widget = (*RadioGroup[string])(nil)

func NewRadioGroup[T AcceptedValue](label string, options []NamedOption[T], setting interactive.StatefulSetting[T]) *RadioGroup[T] {
	selected := setting.Get()

	r := &RadioGroup[T]{
		Options: options,
		Label:   label,

		LastSelected: -1,

		setting: setting,
	}
	r.updateValue(selected, false)
	r.AddLifeCycleFn(r.loop)
	r.ExtendBaseWidget(r)
	return r
}

func (r *RadioGroup[T]) CreateRenderer() fyne.WidgetRenderer {
	r.ExtendBaseWidget(r)

	objects := make([]fyne.CanvasObject, len(r.Options))
	items := make([]*radioItem, len(r.Options))
	for i, option := range r.Options {
		idx := i
		item := newRadioItem(option.Label, func(item *radioItem) {
			r.itemTapped(idx)
		})
		items[idx] = item
		objects[idx] = item
	}

	label := canvas.NewText(r.Label, theme.Color(theme.ColorNamePlaceHolder))
	label.TextSize = 12
	objects = append(objects, label)

	render := &radioGroupRenderer[T]{objects: objects, items: items, label: label, radio: r}

	render.updateItems(false)
	render.Created(r)

	return render
}

//func (r *RadioGroup[T]) MinSize() fyne.Size {
//	r.ExtendBaseWidget(r)
//	return r.BaseWidget.MinSize()
//}

func (r *RadioGroup[T]) SetSelected(index int) {
	if r.SelectedIndex == index {
		return
	}

	err := r.setting.Save(r.Options[index].Value)
	if err != nil {
		// pop up
		dialog.NewError(err, custom_fyne.GetParent()).Show()
		return
	}

	r.SelectedIndex = index

	r.Refresh()
}

func (r *RadioGroup[T]) itemTapped(idx int) {
	if r.Disabled() {
		return
	}

	r.SetSelected(idx)
}

func (r *RadioGroup[T]) loop(stopCh <-chan struct{}) {
	ch := r.setting.Subscribe()
	r.updateValue(r.setting.Get(), true)
	defer ch.Close()

	for {
		select {
		case <-stopCh:
			return
		case value := <-ch.Channel:
			r.updateValue(value, true)
		}
	}
}

func (r *RadioGroup[T]) updateValue(value T, refresh bool) {
	_, index, _ := lo.FindIndexOf(r.Options, func(item NamedOption[T]) bool {
		return item.Value == value
	})

	if r.SelectedIndex != index {
		r.SelectedIndex = index
		if refresh {
			fyne.Do(r.Refresh)
		}
	}
}

type radioGroupRenderer[T AcceptedValue] struct {
	interactive_widgets.BaseLifeCycleRenderer

	objects []fyne.CanvasObject

	label *canvas.Text
	items []*radioItem

	radio *RadioGroup[T]
}

// Layout the components of the radio widget
func (r *radioGroupRenderer[T]) Layout(_ fyne.Size) {
	count := 1
	if len(r.items) > 0 {
		count = len(r.items)
	}
	var itemHeight, itemWidth float32
	minSize := r.radio.MinSize()
	if r.radio.Horizontal {
		itemHeight = minSize.Height
		itemWidth = minSize.Width / float32(count)
	} else {
		itemHeight = minSize.Height / float32(count)
		itemWidth = minSize.Width
	}

	itemSize := fyne.NewSize(itemWidth, itemHeight)
	x, y := float32(0), r.label.MinSize().Height+theme.Padding()
	for _, item := range r.items {
		item.Resize(itemSize)
		item.Move(fyne.NewPos(x, y))
		if r.radio.Horizontal {
			x += itemWidth
		} else {
			y += itemHeight
		}
	}

	r.label.Move(fyne.NewPos(0, 0))
	r.label.Resize(r.label.MinSize())
}

// MinSize calculates the minimum size of a radio item.
// This is based on the contained text, the radio icon and a standard amount of padding
// between each item.
func (r *radioGroupRenderer[T]) MinSize() fyne.Size {
	width := float32(0)
	height := float32(0)
	for _, item := range r.items {
		itemMin := item.MinSize()

		width = fyne.Max(width, itemMin.Width)
		height = fyne.Max(height, itemMin.Height)
	}

	if r.radio.Horizontal {
		width = width * float32(len(r.items))
	} else {
		height = height * float32(len(r.items))
	}

	return fyne.NewSize(width, height+r.label.MinSize().Height+theme.Padding())
}

func (r *radioGroupRenderer[T]) Refresh() {
	r.updateItems(true)
	canvas.Refresh(r.radio)
}

func (r *radioGroupRenderer[T]) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *radioGroupRenderer[T]) updateItems(refresh bool) {
	disable := r.radio.Disabled() && !r.radio.LastDisabled
	enable := !r.radio.Disabled() && r.radio.LastDisabled

	for i, item := range r.items {
		if i == r.radio.SelectedIndex {
			item.SetSelected(true)
		}
		if i == r.radio.LastSelected {
			item.SetSelected(false)
		}
		if disable {
			item.Disable()
		}
		if enable {
			item.Enable()
		}
		if refresh {
			item.Refresh()
		}
	}
}
