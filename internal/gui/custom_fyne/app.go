package custom_fyne

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var a fyne.App
var mainWindow fyne.Window
var countdown *utils.CountdownManager

var AppDataRoot string
var LowAppDataRoot string
var AppConfigRoot string

const AppName = "VRCDP"

func InitRoot() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	AppConfigRoot = filepath.Join(configDir, AppName)
	LowAppDataRoot = AppConfigRoot
	if strings.HasSuffix(configDir, "Roaming") {
		// it's Windows, and we should store large data to Local
		AppDataRoot = filepath.Join(configDir, "..", "Local", AppName)
		LowAppDataRoot = filepath.Join(configDir, "..", "LocalLow", AppName)
	} else {
		AppDataRoot = filepath.Join(AppConfigRoot, "data")
		LowAppDataRoot = AppConfigRoot
	}
}

func InitFyne() {
	a = app.New()
	a.Settings().SetTheme(&cTheme{})
}

func MainLoop() {
	a.Run()
}
func Quit() {
	fyne.Do(a.Quit)
}

func StartCountdown() func() {
	countdown = utils.NewCountdownManager()
	return countdown.Close
}

func CountdownSession(until time.Time) (*utils.CountdownSession, error) {
	return countdown.NewSession(until)
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
