package gui

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type ItemGui struct {
	ps  *song.PreloadedSong
	plg *PlayListGui

	listItem *containers.DynamicListItem

	Progress binding.Float

	Card        *fyne.Container
	Background  *canvas.Rectangle
	ProgressBar *widget.ProgressBar
	ErrorText   *canvas.Text
	StatusText  *canvas.Text
	SizeText    *canvas.Text
	PlayBar     *widgets.PlayBar

	RunningAnimation *fyne.Animation

	StopCh   chan struct{}
	changeCh chan song.ChangeType
}

func NewItemGui(ps *song.PreloadedSong, plg *PlayListGui) *ItemGui {
	info := ps.GetInfo()
	// Title
	title := widgets.NewEllipseText(info.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Favorite button
	favoriteBtn := widgets.NewFavoriteBtn(info.ID, info.Title)

	titleBar := container.NewBorder(nil, nil, nil, favoriteBtn, title)

	// ID
	id := canvas.NewText(info.ID, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	// Group
	group := canvas.NewText(info.Group, theme.Color(theme.ColorNameForeground))
	group.TextSize = 14

	// Adder
	adder := canvas.NewText(info.Adder, theme.Color(theme.ColorNamePlaceHolder))
	adder.TextSize = 14

	// Status
	statusText := canvas.NewText("", theme.Color(theme.ColorNamePlaceHolder))
	statusText.TextSize = 16

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

	// Thumbnail
	thumbnail := widgets.NewThumbnail(info.ThumbnailURL)

	content := NewDynamicFrame(
		thumbnail,
		container.NewVBox(
			group,
			adder,
		),
		container.NewVBox(
			id,
			progressBar,
			sizeText,
		),
		statusText,
		errorText,
	)

	// Play Bar
	playBar := widgets.NewPlayBar()
	playBar.Hide()

	cardContent := container.NewPadded(
		container.NewVBox(
			titleBar,
			content,
			playBar,
		),
	)
	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))
	cardBackground.CornerRadius = theme.Padding() * 2
	cardBackground.StrokeWidth = 2
	cardBackground.StrokeColor = theme.Color(theme.ColorNameSeparator)
	card := container.NewStack(cardBackground, container.NewPadded(cardContent))
	card.Hide()

	ig := ItemGui{
		ps:  ps,
		plg: plg,

		listItem: containers.NewDynamicListItem(ps.GetId(), plg.list, card),

		Progress: bProgress,

		Card:        card,
		Background:  cardBackground,
		ProgressBar: progressBar,
		StatusText:  statusText,
		ErrorText:   errorText,
		SizeText:    sizeText,
		PlayBar:     playBar,

		StopCh:   make(chan struct{}, 10),
		changeCh: ps.SubscribeEvent(),
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

	if status.PreloadError != nil {
		ig.ErrorText.Text = status.PreloadError.Error()
		ig.ErrorText.Refresh()
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
		if !ig.PlayBar.Visible() {
			ig.PlayBar.Show()
			canvas.NewColorRGBAAnimation(
				theme.Color(theme.ColorNameSeparator),
				theme.Color(theme.ColorNamePrimary),
				500*time.Millisecond,
				func(c color.Color) {
					ig.Background.StrokeColor = c
					ig.Background.Refresh()
				},
			).Start()
			ig.listItem.NotifyUpdateMinSize()
		}
	} else {
		if ig.PlayBar.Visible() {
			ig.PlayBar.Hide()
			canvas.NewColorRGBAAnimation(
				theme.Color(theme.ColorNamePrimary),
				theme.Color(theme.ColorNameSeparator),
				500*time.Millisecond,
				func(c color.Color) {
					ig.Background.StrokeColor = c
					ig.Background.Refresh()
				},
			).Start()
			ig.listItem.NotifyUpdateMinSize()
		}
	}
}

func (ig *ItemGui) SlideIn() {
	ig.Card.Move(fyne.NewPos(ig.Card.Size().Width, 0))
	ig.Card.Show()
	ig.RunningAnimation = canvas.NewPositionAnimation(
		fyne.NewPos(ig.Card.Size().Width, 0),
		fyne.NewPos(0, 0),
		300*time.Millisecond,
		ig.Card.Move,
	)
	ig.RunningAnimation.Start()
}
func (ig *ItemGui) SlideOut() {
	if ig.RunningAnimation != nil {
		ig.RunningAnimation.Stop()
	}
	canvas.NewPositionAnimation(
		fyne.NewPos(0, 0),
		fyne.NewPos(-ig.Card.Size().Width, 0),
		300*time.Millisecond,
		ig.Card.Move,
	).Start()
}

func (ig *ItemGui) RenderLoop() {
	for {
		select {
		case <-ig.StopCh:
			return
		case event := <-ig.changeCh:
			switch event {
			case song.StatusChange:
				ig.UpdateStatus()
				switch ig.ps.GetPreloadStatus() {
				case song.Removed:
					ig.SlideOut()
					ig.listItem.MarkRemoving()
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
