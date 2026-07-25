package widgets

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type MultiSelect struct {
	interactive_widgets.LifeCycleWidget

	Options []input.NamedOption[string]

	full []string

	setting interactive.StatefulSetting[[]string]

	OptionSelected []bool

	columnCount, rowCount int
}

func NewMultiSelect(options []input.NamedOption[string], setting interactive.StatefulSetting[[]string]) *MultiSelect {
	m := &MultiSelect{
		Options: options,
		setting: setting,
		full: lo.Map(options, func(o input.NamedOption[string], _ int) string {
			return o.Value
		}),
	}
	m.ExtendBaseWidget(m)
	m.AddLifeCycleFn(m.loop)
	return m
}

var _ fyne.Tappable = (*MultiSelect)(nil)

func (m *MultiSelect) loop(stopCh <-chan struct{}) {
	ch := m.setting.Subscribe()
	defer ch.Close()

	m.update(m.setting.Get())

	for {
		select {
		case <-stopCh:
			return
		case value := <-ch.Channel:
			m.update(value)
		}
	}
}

func (m *MultiSelect) CreateRenderer() fyne.WidgetRenderer {
	m.OptionSelected = make([]bool, len(m.Options))
	selectedOptions := m.setting.Get()
	options := lo.Map(m.Options, func(o input.NamedOption[string], index int) *Option {
		selected := lo.Contains(selectedOptions, o.Value)
		m.OptionSelected[index] = selected
		return NewOption(o.Label, selected)
	})

	r := &MultiSelectRenderer{
		m:       m,
		options: options,
	}
	r.Created(m)

	return r
}

func (m *MultiSelect) Tapped(e *fyne.PointEvent) {
	go func() {
		size := m.Size()
		p := theme.Padding()

		itemWidth := (size.Width + p) / float32(m.columnCount)
		itemHeight := (size.Height + p) / float32(m.rowCount)
		column := int(e.Position.X / itemWidth)
		row := int(e.Position.Y / itemHeight)
		if float32(column+1)*itemWidth-e.Position.X < p || float32(row+1)*itemHeight-e.Position.Y < p {
			return
		}

		index := row*m.columnCount + column
		if index >= len(m.Options) {
			return
		}
		m.OptionSelected[index] = !m.OptionSelected[index]

		values := lo.Filter(m.full, func(_ string, i int) bool {
			return m.OptionSelected[i]
		})

		if err := m.setting.Save(values); err != nil {
			// pop up
			dialog.NewError(err, custom_fyne.GetParent()).Show()
			m.OptionSelected[index] = !m.OptionSelected[index]
			return
		}

		fyne.Do(m.Refresh)
	}()
}

func (m *MultiSelect) update(value []string) {
	for i, option := range m.full {
		m.OptionSelected[i] = lo.Contains(value, option)
	}
	fyne.Do(m.Refresh)
}

type MultiSelectRenderer struct {
	interactive_widgets.BaseLifeCycleRenderer

	m       *MultiSelect
	options []*Option

	minHeight float32
}

func (r *MultiSelectRenderer) getMaxOptionWidth() float32 {
	return lo.Max(
		lo.Map(r.options, func(option *Option, _ int) float32 {
			return option.MinSize().Width
		}),
	)
}

func (r *MultiSelectRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.getMaxOptionWidth(), r.minHeight)
}

func (r *MultiSelectRenderer) Layout(size fyne.Size) {
	if r.options == nil {
		return
	}
	p := theme.Padding()

	columns := min(len(r.options), int((size.Width+p)/(r.getMaxOptionWidth()+p)))
	rows := (len(r.options) + columns - 1) / columns

	r.m.columnCount = columns
	r.m.rowCount = rows

	itemWidth := (size.Width+p)/float32(columns) - p

	accY := p
	for i := 0; i < rows; i++ {
		itemCount := min(columns, len(r.options)-i*columns)
		rowItems := r.options[i*columns : i*columns+itemCount]
		maxItemHeight := lo.Max(
			lo.Map(rowItems, func(option *Option, _ int) float32 {
				return option.MinSize().Height
			}),
		)
		for j, option := range rowItems {
			option.Resize(fyne.NewSize(itemWidth, maxItemHeight))
			option.Move(fyne.NewPos((itemWidth+p)*float32(j), accY))
		}
		accY += maxItemHeight + p
	}
	accY -= p

	if math.Abs(float64(accY-r.minHeight)) > 1e-1 {
		r.minHeight = accY
		r.m.Refresh()
	}
}

func (r *MultiSelectRenderer) Objects() []fyne.CanvasObject {
	return lo.Map(r.options, func(option *Option, _ int) fyne.CanvasObject {
		return option
	})
}

func (r *MultiSelectRenderer) Refresh() {
	for i, item := range r.options {
		item.SetSelected(r.m.OptionSelected[i])
	}
}

type Option struct {
	widget.BaseWidget

	Label    string
	Selected bool
}

func NewOption(label string, selected bool) *Option {
	o := &Option{
		Label:    label,
		Selected: selected,
	}
	o.ExtendBaseWidget(o)
	return o
}

func (o *Option) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText(o.Label, theme.Color(theme.ColorNameForeground))
	label.Alignment = fyne.TextAlignCenter
	rect := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	rect.CornerRadius = theme.Padding()
	check := widget.NewIcon(theme.NewColoredResource(theme.ConfirmIcon(), theme.ColorNameForegroundOnPrimary))

	if o.Selected {
		label.Color = color.White
		rect.FillColor = theme.Color(theme.ColorNamePrimary)
	} else {
		check.Hide()
	}

	return &OptionRenderer{
		o:     o,
		label: label,
		rect:  rect,
		check: check,
	}
}

func (o *Option) SetSelected(selected bool) {
	if selected == o.Selected {
		return
	}
	o.Selected = selected
	fyne.Do(o.Refresh)
}

type OptionRenderer struct {
	o *Option

	label *canvas.Text
	rect  *canvas.Rectangle
	check *widget.Icon
}

func (o *OptionRenderer) MinSize() fyne.Size {
	labelSize := o.label.MinSize()
	p := theme.Padding()
	return fyne.NewSize(labelSize.Width+p*2, labelSize.Height+p*2)
}

func (o *OptionRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	o.label.Resize(fyne.NewSize(size.Width-p*2, size.Height-p*2))
	o.label.Move(fyne.NewPos(p, p))
	o.rect.Resize(size)
	o.rect.Move(fyne.NewPos(0, 0))
	if o.check.Visible() {
		o.check.Move(fyne.NewPos(0, 0))
		o.check.Resize(fyne.NewSquareSize(12))
	}
}

func (o *OptionRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{o.rect, o.label, o.check}
}

func (o *OptionRenderer) Refresh() {
	if o.o.Selected {
		o.label.Color = color.White
		o.rect.FillColor = theme.Color(theme.ColorNamePrimary)
		o.check.Show()
	} else {
		o.label.Color = theme.Color(theme.ColorNameForeground)
		o.rect.FillColor = theme.Color(theme.ColorNameInputBackground)
		o.check.Hide()
	}
	canvas.Refresh(o.o)
}

func (o *OptionRenderer) Destroy() {}
