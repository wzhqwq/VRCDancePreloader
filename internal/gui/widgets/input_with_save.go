package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
)

type InputWithSave struct {
	widget.BaseWidget
	Value string

	Label       *canvas.Text
	InputWidget *widget.Entry
	SaveBtn     *widget.Button

	InputAppendItems []fyne.CanvasObject
	AfterSaveItems   []fyne.CanvasObject

	ForceDigits bool
	Dirty       bool

	OnSave func()
}

func NewInputWithSave(value, label string) *InputWithSave {
	t := &InputWithSave{}

	t.Extend(value, label)

	t.ExtendBaseWidget(t)

	return t
}

func (i *InputWithSave) Extend(value, label string) {
	i.Value = value

	i.Label = canvas.NewText(label, theme.Color(theme.ColorNamePlaceHolder))
	i.Label.Text = label
	i.Label.TextSize = 12

	i.InputWidget = widget.NewEntry()
	i.InputWidget.SetText(value)
	i.InputWidget.Wrapping = fyne.TextWrapOff
	i.InputWidget.Scroll = container.ScrollNone
	i.InputWidget.Refresh()
	i.InputWidget.OnChanged = func(s string) {
		newDirty := strings.Compare(i.Value, s) != 0
		if newDirty != i.Dirty {
			i.Dirty = newDirty
			if i.Dirty {
				i.SaveBtn.Show()
				for _, item := range i.AfterSaveItems {
					item.Hide()
				}
			} else {
				i.SaveBtn.Hide()
				for _, item := range i.AfterSaveItems {
					item.Show()
				}
			}
		}
	}
	i.InputWidget.OnSubmitted = func(s string) {
		i.SetValue(i.InputWidget.Text)
	}

	i.SaveBtn = widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		i.SetValue(i.InputWidget.Text)
	})
	i.SaveBtn.Importance = widget.HighImportance
	i.SaveBtn.Hide()
}

func (i *InputWithSave) SetValue(value string) {
	if value == i.Value {
		return
	}

	value = strings.Trim(value, " ")
	if i.ForceDigits {
		// remove non-digits
		value = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, value)
	}

	i.Value = value
	i.InputWidget.SetText(value)
	if i.OnSave != nil {
		i.OnSave()
	}
	i.Dirty = false

	i.SaveBtn.Hide()
	for _, item := range i.AfterSaveItems {
		item.Show()
	}
	i.Refresh()
}

func (i *InputWithSave) CreateRenderer() fyne.WidgetRenderer {
	return &inputWithSaveRenderer{
		i: i,
	}
}

type inputWithSaveRenderer struct {
	i *InputWithSave
}

func (r *inputWithSaveRenderer) MinSize() fyne.Size {
	return fyne.NewSize(100, r.i.Label.MinSize().Height+r.i.InputWidget.MinSize().Height+theme.Padding()*2)
}

func (r *inputWithSaveRenderer) Layout(size fyne.Size) {
	r.i.Label.Move(fyne.NewPos(theme.Padding(), 0))

	labelHeight := r.i.Label.MinSize().Height + theme.Padding()
	inputHeight := size.Height - theme.Padding() - labelHeight
	buttonY := labelHeight + inputHeight/2
	rightWidth := theme.Padding()

	if r.i.Dirty {
		rightWidth += r.i.SaveBtn.MinSize().Width
		r.i.SaveBtn.Move(fyne.NewPos(size.Width-rightWidth, buttonY-r.i.SaveBtn.MinSize().Height/2))
		r.i.SaveBtn.Resize(r.i.SaveBtn.MinSize())
		rightWidth += theme.Padding()
	} else {
		for i := len(r.i.AfterSaveItems) - 1; i >= 0; i-- {
			item := r.i.AfterSaveItems[i]
			rightWidth += item.MinSize().Width
			item.Move(fyne.NewPos(size.Width-rightWidth, buttonY-item.MinSize().Height/2))
			item.Resize(item.MinSize())
			rightWidth += theme.Padding()
		}
	}

	appendItemRight := rightWidth + theme.Padding()

	for i := len(r.i.InputAppendItems) - 1; i >= 0; i-- {
		item := r.i.InputAppendItems[i]
		appendItemRight += item.MinSize().Width
		item.Move(fyne.NewPos(size.Width-appendItemRight, buttonY-item.MinSize().Height/2))
		item.Resize(item.MinSize())
		appendItemRight += theme.Padding()
	}

	r.i.InputWidget.Move(fyne.NewPos(theme.Padding(), labelHeight))
	r.i.InputWidget.Resize(fyne.NewSize(size.Width-theme.Padding()-rightWidth, inputHeight))
}

func (r *inputWithSaveRenderer) Refresh() {
	r.Layout(r.i.Size())
}

func (r *inputWithSaveRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.i.Label, r.i.InputWidget, r.i.SaveBtn}
	objects = append(objects, r.i.InputAppendItems...)
	objects = append(objects, r.i.AfterSaveItems...)
	return objects
}

func (r *inputWithSaveRenderer) Destroy() {
}
