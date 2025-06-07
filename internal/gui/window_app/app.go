package window_app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

var a fyne.App
var mainWindow fyne.Window

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

func NewMainWindow(title string) fyne.Window {
	mainWindow = a.NewWindow(title)
	mainWindow.SetMaster()
	return mainWindow
}

func NewWindow(title string) fyne.Window {
	return a.NewWindow(title)
}

func GetParent() fyne.Window {
	return mainWindow
}

func Driver() fyne.Driver {
	return a.Driver()
}
