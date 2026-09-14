package input

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func WrapLabel(content fyne.CanvasObject, label string) *fyne.Container {
	t := canvas.NewText(label, theme.Color(theme.ColorNamePlaceHolder))
	t.TextSize = 12
	return container.NewVBox(t, content)
}
