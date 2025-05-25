package window_app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

var a fyne.App

func InitFyne() {
	a = app.New()
	a.Settings().SetTheme(&cTheme{})
}

func MainLoop() {
	a.Run()
}
func Quit() {
	fyne.Do(func() {
		a.Quit()
	})
}

func NewWindow(title string) fyne.Window {
	return a.NewWindow(title)
}

func Driver() fyne.Driver {
	return a.Driver()
}
