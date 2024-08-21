package gui

import (
	"os"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var a fyne.App = app.New()

func MainWindow() {
	w := a.NewWindow("Hello")

	hello := widget.NewLabel("Hello Fyne!")
	w.SetContent(container.NewVBox(
		hello,
		widget.NewButton("Hi!", func() {
			hello.SetText("Welcome :)")
		}),
	))

	w.Show()
}

func MainLoop(closeCh chan os.Signal) {
	a.Run()
	closeCh <- syscall.SIGINT
}
