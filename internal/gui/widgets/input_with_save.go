package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"log"
)

type InputWithSave struct {
	widget.BaseWidget
	Value string

	Label       *canvas.Text
	InputWidget *widget.Entry
	SaveBtn     *widget.Button

	BeforeSaveItems []fyne.CanvasObject
	AfterSaveItems  []fyne.CanvasObject

	ForceDigits bool

	OnSave func()
}

func NewInputWithSave(value, label string) *InputWithSave {
	inputWidget := widget.NewEntry()
	inputWidget.SetText(value)
	inputWidget.Wrapping = fyne.TextWrapOff
	inputWidget.Scroll = container.ScrollNone
	inputWidget.Refresh()
	labelText := canvas.NewText(label, theme.Color(theme.ColorNamePlaceHolder))
	labelText.TextSize = 12

	t := &InputWithSave{
		Value: value,

		Label:       labelText,
		InputWidget: inputWidget,

		OnSave: func() {},
	}

	t.SaveBtn = widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		log.Println("Save")
		t.SetValue(inputWidget.Text)
	})

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

	i.SaveBtn = widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		log.Println("Save")
		i.SetValue(i.InputWidget.Text)
	})
}

func (i *InputWithSave) SetValue(value string) {
	if value == i.Value {
		return
	}
	i.Value = value
	i.OnSave()
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

	for i := len(r.i.AfterSaveItems) - 1; i >= 0; i-- {
		item := r.i.AfterSaveItems[i]
		rightWidth += item.MinSize().Width
		item.Move(fyne.NewPos(size.Width-rightWidth, buttonY-item.MinSize().Height/2))
		item.Resize(item.MinSize())
		rightWidth += theme.Padding()
	}

	rightWidth += r.i.SaveBtn.MinSize().Width
	r.i.SaveBtn.Move(fyne.NewPos(size.Width-rightWidth, buttonY-r.i.SaveBtn.MinSize().Height/2))
	r.i.SaveBtn.Resize(r.i.SaveBtn.MinSize())
	rightWidth += theme.Padding()

	rightWidthBefore := rightWidth + theme.Padding()

	for i := len(r.i.BeforeSaveItems) - 1; i >= 0; i-- {
		item := r.i.BeforeSaveItems[i]
		rightWidthBefore += item.MinSize().Width
		item.Move(fyne.NewPos(size.Width-rightWidthBefore, buttonY-item.MinSize().Height/2))
		item.Resize(item.MinSize())
		rightWidthBefore += theme.Padding()
	}

	r.i.InputWidget.Move(fyne.NewPos(theme.Padding(), labelHeight))
	r.i.InputWidget.Resize(fyne.NewSize(size.Width-theme.Padding()-rightWidth, inputHeight))
}

func (r *inputWithSaveRenderer) Refresh() {
	r.i.InputWidget.SetText(r.i.Value)
	r.i.InputWidget.Refresh()
	r.i.SaveBtn.Refresh()
}

func (r *inputWithSaveRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.i.Label, r.i.InputWidget, r.i.SaveBtn}
	objects = append(objects, r.i.BeforeSaveItems...)
	objects = append(objects, r.i.AfterSaveItems...)
	return objects
}

func (r *inputWithSaveRenderer) Destroy() {
}
