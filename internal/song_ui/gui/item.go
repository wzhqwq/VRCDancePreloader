package gui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
)

type ItemGui struct {
	ps  *song.PreloadedSong
	plg *PlayListGui

	Progress binding.Float

	Card        *fyne.Container
	ProgressBar *widget.ProgressBar
	ErrorText   *canvas.Text
	StatusText  *canvas.Text
	SizeText    *canvas.Text
	PlayBar     *widgets.PlayBar

	RunningAnimation *fyne.Animation

	StopCh chan struct{}
}

func NewItemGui(ps *song.PreloadedSong, plg *PlayListGui) *ItemGui {
	info := ps.GetInfo()
	// Title
	title := widgets.NewEllipseText(info.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	// ID
	id := canvas.NewText(info.ID, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	// Group
	group := canvas.NewText(info.Group, theme.Color(theme.ColorNameForeground))
	group.TextSize = 12

	// Adder
	adder := canvas.NewText(info.Adder, theme.Color(theme.ColorNamePlaceHolder))
	adder.TextSize = 12

	// Status
	statusText := canvas.NewText("", theme.Color(theme.ColorNamePlaceHolder))
	statusText.TextSize = 14

	// Progress bar
	bProgress := binding.NewFloat()
	progressBar := widget.NewProgressBarWithData(bProgress)

	// Size
	sizeText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	sizeText.Alignment = fyne.TextAlignTrailing
	sizeText.TextSize = 12

	// Error message
	errorText := canvas.NewText("", theme.Color(theme.ColorNameError))
	errorText.TextSize = 12

	// Play Bar
	playBar := widgets.NewPlayBar()

	// Thumbnail
	thumbnail := widgets.NewThumbnail(info.ThumbnailURL)

	cardContent := container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			nil,
			id,
			title,
		),
		NewDynamicFrame(
			thumbnail,
			container.NewBorder(
				nil,
				nil,
				container.NewVBox(
					group,
					adder,
				),
				nil,
				container.NewPadded(
					playBar,
				),
			),
			container.NewVBox(
				progressBar,
				sizeText,
			),
			statusText,
			errorText,
		),
	)
	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))
	cardBackground.CornerRadius = theme.Padding() * 2
	card := container.NewStack(cardBackground, container.NewPadded(cardContent))

	ig := ItemGui{
		ps:  ps,
		plg: plg,

		Progress: bProgress,

		Card:        card,
		ProgressBar: progressBar,
		StatusText:  statusText,
		ErrorText:   errorText,
		SizeText:    sizeText,
		PlayBar:     playBar,

		StopCh: make(chan struct{}),
	}
	ig.UpdateStatus()
	ig.UpdateProgress()
	ig.UpdateTime()

	return &ig
}

func (ig *ItemGui) UpdateStatus() {
	status := ig.ps.GetStatusInfo()

	ig.StatusText.Text = status.Status
	ig.StatusText.Color = theme.Color(status.Color)
	ig.StatusText.Refresh()

	ig.ErrorText.Text = status.PreloadError.Error()
	ig.ErrorText.Refresh()

	if status.PreloadError != nil {
		ig.ErrorText.Show()
	} else {
		ig.ErrorText.Hide()
	}
}
func (ig *ItemGui) UpdateProgress() {
	progress := ig.ps.GetProgressInfo()

	ig.Progress.Set(progress.Progress)
	ig.SizeText.Text = progress.DownloadedPretty
	ig.SizeText.Refresh()

	if progress.IsDownloading {
		ig.ProgressBar.Show()
	} else {
		ig.ProgressBar.Hide()
	}
}
func (ig *ItemGui) UpdateTime() {
	timeInfo := ig.ps.GetTimeInfo()

	ig.PlayBar.Progress = float32(timeInfo.Progress)
	ig.PlayBar.Text = timeInfo.Text
	ig.PlayBar.Refresh()

	if timeInfo.IsPlaying {
		ig.PlayBar.Show()
	} else {
		ig.PlayBar.Hide()
	}
}

func (ig *ItemGui) SlideIn() {
	ig.RunningAnimation = canvas.NewPositionAnimation(
		fyne.NewPos(ig.Card.Size().Width, ig.Card.Position().Y),
		ig.Card.Position(),
		300*time.Millisecond,
		func(pos fyne.Position) {
			ig.Card.Move(pos)
		},
	)
	ig.RunningAnimation.Start()
}
func (ig *ItemGui) SlideOut() {
	if ig.RunningAnimation != nil {
		ig.RunningAnimation.Stop()
	}
	canvas.NewPositionAnimation(
		ig.Card.Position(),
		fyne.NewPos(-ig.Card.Size().Width, ig.Card.Position().Y),
		300*time.Millisecond,
		ig.Card.Move,
	).Start()
}

func (ig *ItemGui) RenderLoop() {
	ch := ig.ps.SubscribeEvent()
	for {
		select {
		case <-ig.StopCh:
			return
		case event := <-ch:
			switch event {
			case song.StatusChange:
				ig.UpdateStatus()
				switch ig.ps.GetPreloadStatus() {
				case song.Removed:
					ig.SlideOut()
					go func() {
						time.Sleep(300 * time.Millisecond)
						ig.plg.removeFromMap(ig.ps.GetId())
					}()
					return
				}
			case song.ProgressChange:
				ig.UpdateProgress()
			case song.TimeChange:
				ig.UpdateTime()
			}
		}
	}
}
