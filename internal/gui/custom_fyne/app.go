package custom_fyne

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

var a fyne.App
var mainWindow fyne.Window

var AppDataRoot string
var AppConfigRoot string

const AppName = "VRCDP"

func InitFyne() {
	a = app.New()
	a.Settings().SetTheme(&cTheme{})

	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	AppConfigRoot = filepath.Join(configDir, AppName)
	if strings.HasSuffix(configDir, "Roaming") {
		// it's Windows, and we should store large data to LocalLow
		AppDataRoot = filepath.Join(configDir, "..", "LocalLow", AppName)
	} else {
		AppDataRoot = filepath.Join(AppConfigRoot, "data")
	}
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

func GetParent() fyne.Window {
	return mainWindow
}

func Driver() fyne.Driver {
	return a.Driver()
}
