package gui

import (
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type ItemGui struct {
	widget.BaseWidget

	ps  *song.PreloadedSong
	plg *PlayListGui

	listItem *containers.DynamicListItem

	// static UI
	Background *canvas.Rectangle
	InfoLeft   fyne.CanvasObject
	InfoRight  fyne.CanvasObject
	InfoBottom fyne.CanvasObject

	ProgressBar *widgets.SizeProgressBar
	ErrorText   *canvas.Text
	StatusText  *canvas.Text
	SizeText    *canvas.Text
	PlayBar     *widgets.PlayBar
	FavoriteBtn *widgets.FavoriteBtn
	TitleWidget *widgets.EllipseText
	Thumbnail   *widgets.Thumbnail

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

	// Group
	group := canvas.NewText(info.Group, theme.Color(theme.ColorNameForeground))
	group.TextSize = 14

	// Adder
	adder := canvas.NewText(info.Adder, theme.Color(theme.ColorNamePlaceHolder))
	adder.TextSize = 14

	// ID
	id := canvas.NewText(info.ID, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	// Progress
	progress := widgets.NewSizeProgressBar(0, 0)
	progress.Text.TextSize = 10

	// Size
	sizeText := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	sizeText.Alignment = fyne.TextAlignTrailing
	sizeText.TextSize = 12

	// Status
	statusText := canvas.NewText("", theme.Color(theme.ColorNamePlaceHolder))
	statusText.TextSize = 16

	// Error message
	errorText := canvas.NewText("", theme.Color(theme.ColorNameError))
	errorText.TextSize = 12

	// Play bar
	playBar := widgets.NewPlayBar()

	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))
	cardBackground.CornerRadius = theme.Padding() * 2
	cardBackground.StrokeWidth = 2
	cardBackground.StrokeColor = theme.Color(theme.ColorNameSeparator)

	ig := &ItemGui{
		ps:  ps,
		plg: plg,

		Background: cardBackground,
		InfoLeft:   container.NewVBox(group, adder, statusText),
		InfoRight:  container.NewVBox(id, progress, sizeText),
		InfoBottom: container.NewVBox(errorText, playBar),

		ProgressBar: progress,
		StatusText:  statusText,
		ErrorText:   errorText,
		SizeText:    sizeText,
		PlayBar:     playBar,
		FavoriteBtn: widgets.NewFavoriteBtn(info.ID, info.Title),
		Thumbnail:   widgets.NewThumbnail(info.ThumbnailURL),
		TitleWidget: title,

		StopCh:   make(chan struct{}, 10),
		changeCh: ps.SubscribeEvent(),
	}
	ig.listItem = containers.NewDynamicListItem(ps.GetId(), plg.list, ig)

	ig.ExtendBaseWidget(ig)

	ig.UpdateStatus()
	ig.UpdateProgress()
	ig.UpdateTime(false)

	return ig
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

	ig.ProgressBar.SetTotalSize(progress.Total)
	ig.ProgressBar.SetCurrentSize(progress.Downloaded)
	if progress.Total > 0 {
		ig.SizeText.Text = utils.PrettyByteSize(progress.Total)
	} else {
		ig.SizeText.Text = i18n.T("placeholder_unknown_size")
	}
	ig.SizeText.Refresh()

	if progress.IsDownloading {
		ig.ProgressBar.Show()
		ig.SizeText.Hide()
	} else {
		ig.ProgressBar.Hide()
		ig.SizeText.Show()
	}
}
func (ig *ItemGui) UpdateTime(animation bool) {
	timeInfo := ig.ps.GetTimeInfo()

	if timeInfo.IsPlaying {
		ig.PlayBar.Progress = float32(timeInfo.Progress)
		ig.PlayBar.Text = timeInfo.Text
		ig.PlayBar.Refresh()
		if !ig.PlayBar.Visible() {
			ig.PlayBar.Show()
			if animation {
				canvas.NewColorRGBAAnimation(
					theme.Color(theme.ColorNameSeparator),
					theme.Color(theme.ColorNamePrimary),
					500*time.Millisecond,
					func(c color.Color) {
						ig.Background.StrokeColor = c
						ig.Background.Refresh()
					},
				).Start()
			} else {
				ig.Background.StrokeColor = theme.Color(theme.ColorNamePrimary)
				ig.Background.Refresh()
			}
			ig.listItem.NotifyUpdateMinSize()
		}
	} else {
		if ig.PlayBar.Visible() {
			ig.PlayBar.Hide()
			if animation {
				canvas.NewColorRGBAAnimation(
					theme.Color(theme.ColorNamePrimary),
					theme.Color(theme.ColorNameSeparator),
					500*time.Millisecond,
					func(c color.Color) {
						ig.Background.StrokeColor = c
						ig.Background.Refresh()
					},
				).Start()
			} else {
				ig.Background.StrokeColor = theme.Color(theme.ColorNameSeparator)
				ig.Background.Refresh()
			}
			ig.listItem.NotifyUpdateMinSize()
		}
	}
}

func (ig *ItemGui) SlideIn() {
	ig.Move(fyne.NewPos(ig.Size().Width, 0))
	ig.Show()
	ig.RunningAnimation = canvas.NewPositionAnimation(
		fyne.NewPos(ig.Size().Width, 0),
		fyne.NewPos(0, 0),
		300*time.Millisecond,
		ig.Move,
	)
	ig.RunningAnimation.Start()
	ig.Refresh()
}
func (ig *ItemGui) SlideOut() {
	if ig.RunningAnimation != nil {
		ig.RunningAnimation.Stop()
	}
	canvas.NewPositionAnimation(
		fyne.NewPos(0, 0),
		fyne.NewPos(-ig.Size().Width, 0),
		300*time.Millisecond,
		ig.Move,
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
				ig.UpdateTime(true)
			}
		}
	}
}

func (ig *ItemGui) CreateRenderer() fyne.WidgetRenderer {
	return &ItemRenderer{
		ig: ig,
	}
}

var playItemMinWidth float32 = 260
var playItemThumbnailShow float32 = 300
var playItemThumbnailStartWidth float32 = 60
var playItemThumbnailMaxWidth float32 = 120
var topHeight float32 = 30

type ItemRenderer struct {
	ig *ItemGui
}

func (r *ItemRenderer) MinSize() fyne.Size {
	p := theme.Padding()
	totalHeight := topHeight + p
	totalHeight += r.ig.InfoLeft.MinSize().Height + p
	bottomHeight := r.ig.InfoBottom.MinSize().Height
	if bottomHeight > 0 {
		totalHeight += bottomHeight + p
	}
	return fyne.NewSize(playItemMinWidth, totalHeight)
}

func (r *ItemRenderer) Layout(size fyne.Size) {
	r.ig.Background.Resize(size)
	r.ig.Background.Move(fyne.NewPos(0, 0))
	p := theme.Padding()

	// Top layout: title and favorite button
	btnSize := r.ig.FavoriteBtn.MinSize().Height
	btnP := (topHeight - btnSize) / 2
	favoriteX := size.Width - btnSize - btnP
	r.ig.FavoriteBtn.Resize(fyne.NewSize(btnSize, btnSize))
	r.ig.FavoriteBtn.Move(fyne.NewPos(favoriteX, btnP))

	titleWidth := favoriteX - p - btnP
	titleHeight := r.ig.TitleWidget.MinSize().Height
	r.ig.TitleWidget.Resize(fyne.NewSize(titleWidth, titleHeight))
	r.ig.TitleWidget.Move(fyne.NewPos(p, (topHeight-titleHeight)/2))

	// Bottom layout: play bar or error message
	bottomHeight := r.ig.InfoBottom.MinSize().Height
	bottomY := size.Height - bottomHeight - p
	r.ig.InfoBottom.Resize(fyne.NewSize(size.Width-p*2, bottomHeight))
	r.ig.InfoBottom.Move(fyne.NewPos(p, bottomY))

	// Center layout
	centerY := topHeight + p
	centerHeight := bottomY - centerY

	// right layout: id, progress, size
	rightInfoX := size.Width - r.ig.InfoRight.MinSize().Width - p
	r.ig.InfoRight.Resize(fyne.NewSize(r.ig.InfoRight.MinSize().Width, centerHeight))
	r.ig.InfoRight.Move(fyne.NewPos(rightInfoX, topHeight))

	// left layout: thumbnail(if possible), group, adder, status
	leftInfoX := p
	if size.Width > playItemThumbnailShow {
		thumbnailWidth := playItemThumbnailStartWidth + size.Width - playItemThumbnailShow
		if thumbnailWidth > playItemThumbnailMaxWidth {
			thumbnailWidth = playItemThumbnailMaxWidth
		}

		r.ig.Thumbnail.Resize(fyne.NewSize(thumbnailWidth, centerHeight))
		r.ig.Thumbnail.Move(fyne.NewPos(leftInfoX, topHeight))

		leftInfoX += thumbnailWidth + p
	} else {
		r.ig.Thumbnail.Resize(fyne.NewSize(0, centerHeight))
	}

	r.ig.InfoLeft.Resize(fyne.NewSize(rightInfoX-leftInfoX, centerHeight))
	r.ig.InfoLeft.Move(fyne.NewPos(leftInfoX, topHeight))
}

func (r *ItemRenderer) Refresh() {

}

func (r *ItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.ig.Background,

		r.ig.TitleWidget,
		r.ig.FavoriteBtn,

		r.ig.InfoBottom,

		r.ig.Thumbnail,
		r.ig.InfoLeft,
		r.ig.InfoRight,
	}
}

func (r *ItemRenderer) Destroy() {
	r.ig.StopCh <- struct{}{}
}
