package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"image/color"
	"math"
)

type MultiSelect struct {
	widget.BaseWidget

	Options []string
	Values  []string

	OptionSelected []bool

	OnChange func([]string)
}

func NewMultiSelect(options, values []string) *MultiSelect {
	m := &MultiSelect{
		Options: options,
		Values:  values,
	}
	m.ExtendBaseWidget(m)
	return m
}

func (m *MultiSelect) CreateRenderer() fyne.WidgetRenderer {
	m.OptionSelected = make([]bool, len(m.Options))
	options := lo.Map(m.Options, func(value string, index int) *Option {
		selected := lo.IndexOf(m.Values, value) >= 0
		m.OptionSelected[index] = selected
		return NewOption(value, selected, func(s bool) {
			m.OptionSelected[index] = s
			m.update()
		})
	})

	return &MultiSelectRenderer{
		m:       m,
		options: options,
	}
}

func (m *MultiSelect) update() {
	m.Values = lo.Filter(m.Options, func(_ string, index int) bool {
		return m.OptionSelected[index]
	})
	if m.OnChange != nil {
		m.OnChange(m.Values)
	}
}

type MultiSelectRenderer struct {
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
	return fyne.NewSize(r.getMaxOptionWidth()+theme.Padding()*2, r.minHeight)
}

func (r *MultiSelectRenderer) Layout(size fyne.Size) {
	if r.options == nil {
		return
	}
	p := theme.Padding()

	columns := min(len(r.options), int(size.Width/(r.getMaxOptionWidth()+p)))
	rows := (len(r.options) + columns - 1) / columns

	itemWidth := (size.Width-p)/float32(columns) - p

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
			option.Move(fyne.NewPos(p+(itemWidth+p)*float32(j), accY))
		}
		accY += maxItemHeight + p
	}

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
	canvas.Refresh(r.m)
}

func (r *MultiSelectRenderer) Destroy() {}

type Option struct {
	widget.BaseWidget
	fyne.Tappable

	Label    string
	Selected bool

	OnChange func(bool)
}

func NewOption(label string, selected bool, onChange func(bool)) *Option {
	o := &Option{
		Label:    label,
		Selected: selected,
		OnChange: onChange,
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

func (o *Option) Tapped(e *fyne.PointEvent) {
	o.Selected = !o.Selected
	if o.OnChange != nil {
		o.OnChange(o.Selected)
	}
	o.Refresh()
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
}

func (o *OptionRenderer) Destroy() {}
