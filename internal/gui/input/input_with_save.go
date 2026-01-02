package input

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
)

type InputWithSave struct {
	widget.BaseWidget
	Value string
	Label string

	InputAppendItems []fyne.CanvasObject
	AfterSaveItems   []fyne.CanvasObject

	ForceDigits bool
	Dirty       bool

	OnSave func() error
}

func NewInputWithSave(value, label string) *InputWithSave {
	t := &InputWithSave{}

	t.Extend(value, label)

	t.ExtendBaseWidget(t)

	return t
}

func (i *InputWithSave) Extend(value, label string) {
	i.Value = value
	i.Label = label
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
	if i.OnSave != nil {
		err := i.OnSave()
		if err != nil {
			// pop up
			dialog.NewError(err, custom_fyne.GetParent()).Show()
			return
		}
	}
	i.Dirty = false

	fyne.Do(func() {
		i.Refresh()
	})
}

func (i *InputWithSave) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText(i.Label, theme.Color(theme.ColorNamePlaceHolder))
	label.TextSize = 12

	input := widget.NewEntry()

	saveBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		i.SetValue(input.Text)
	})
	saveBtn.Importance = widget.HighImportance
	saveBtn.Hide()

	input.SetText(i.Value)
	input.Wrapping = fyne.TextWrapOff
	input.Scroll = container.ScrollNone
	input.Refresh()
	input.OnChanged = func(s string) {
		newDirty := strings.Compare(i.Value, s) != 0
		if newDirty != i.Dirty {
			i.Dirty = newDirty
			i.Refresh()
		}
	}
	input.OnSubmitted = func(s string) {
		i.SetValue(input.Text)
	}

	return &inputWithSaveRenderer{
		i: i,

		Label:       label,
		InputWidget: input,
		SaveBtn:     saveBtn,
	}
}

type inputWithSaveRenderer struct {
	i *InputWithSave

	Label       *canvas.Text
	InputWidget *widget.Entry
	SaveBtn     *widget.Button
}

func (r *inputWithSaveRenderer) MinSize() fyne.Size {
	return fyne.NewSize(100, r.Label.MinSize().Height+r.InputWidget.MinSize().Height+theme.Padding())
}

func (r *inputWithSaveRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	r.Label.Move(fyne.NewPos(0, p/2))

	labelHeight := r.Label.MinSize().Height + p
	inputHeight := size.Height - labelHeight
	buttonY := labelHeight + inputHeight/2
	rightWidth := float32(0)

	if r.i.Dirty {
		rightWidth += r.SaveBtn.MinSize().Width
		r.SaveBtn.Move(fyne.NewPos(size.Width-rightWidth, buttonY-r.SaveBtn.MinSize().Height/2))
		r.SaveBtn.Resize(r.SaveBtn.MinSize())
		rightWidth += p
	} else {
		for i := len(r.i.AfterSaveItems) - 1; i >= 0; i-- {
			item := r.i.AfterSaveItems[i]
			rightWidth += item.MinSize().Width
			item.Move(fyne.NewPos(size.Width-rightWidth, buttonY-item.MinSize().Height/2))
			item.Resize(item.MinSize())
			rightWidth += p
		}
	}

	appendItemRight := rightWidth + p

	for i := len(r.i.InputAppendItems) - 1; i >= 0; i-- {
		item := r.i.InputAppendItems[i]
		appendItemRight += item.MinSize().Width
		item.Move(fyne.NewPos(size.Width-appendItemRight, buttonY-item.MinSize().Height/2))
		item.Resize(item.MinSize())
		appendItemRight += p
	}

	r.InputWidget.Move(fyne.NewPos(0, labelHeight))
	r.InputWidget.Resize(fyne.NewSize(size.Width-rightWidth, inputHeight))
}

func (r *inputWithSaveRenderer) Refresh() {
	if r.i.Dirty {
		r.SaveBtn.Show()
		for _, item := range r.i.AfterSaveItems {
			item.Hide()
		}
	} else {
		r.InputWidget.SetText(r.i.Value)
		r.SaveBtn.Hide()
		for _, item := range r.i.AfterSaveItems {
			item.Show()
		}
	}

	canvas.Refresh(r.i)
}

func (r *inputWithSaveRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.Label, r.InputWidget, r.SaveBtn}
	objects = append(objects, r.i.InputAppendItems...)
	objects = append(objects, r.i.AfterSaveItems...)
	return objects
}

func (r *inputWithSaveRenderer) Destroy() {
}
