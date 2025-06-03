package playlist

import (
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
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

	ps *song.PreloadedSong
	dl *containers.DynamicList

	listItem *containers.DynamicListItem

	// static UI

	RunningAnimation *fyne.Animation

	StopCh   chan struct{}
	changeCh chan song.ChangeType

	statusChanged   bool
	timeChanged     bool
	progressChanged bool
}

func NewItemGui(ps *song.PreloadedSong, dl *containers.DynamicList) *ItemGui {
	ig := &ItemGui{
		ps: ps,
		dl: dl,

		StopCh:   make(chan struct{}, 10),
		changeCh: ps.SubscribeEvent(),

		statusChanged:   true,
		timeChanged:     true,
		progressChanged: true,
	}
	ig.listItem = containers.NewDynamicListItem(ps.GetId(), dl, ig)

	ig.ExtendBaseWidget(ig)

	return ig
}

func (ig *ItemGui) SlideIn() {
	fyne.Do(func() {
		ig.Move(fyne.NewPos(ig.Size().Width, 0))
		ig.Show()
		ig.RunningAnimation = canvas.NewPositionAnimation(
			fyne.NewPos(ig.Size().Width, 0),
			fyne.NewPos(0, 0),
			300*time.Millisecond,
			ig.Move,
		)
		ig.RunningAnimation.Start()
	})
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
				ig.statusChanged = true
				switch ig.ps.GetPreloadStatus() {
				case song.Removed:
					ig.SlideOut()
					ig.listItem.MarkRemoving()
					go func() {
						time.Sleep(300 * time.Millisecond)
						ig.dl.RemoveItem(ig.ps.GetId())
					}()
					return
				}
			case song.ProgressChange:
				ig.progressChanged = true
			case song.TimeChange:
				ig.timeChanged = true
			}
			fyne.Do(func() {
				ig.Refresh()
			})
		}
	}
}

func (ig *ItemGui) CreateRenderer() fyne.WidgetRenderer {
	info := ig.ps.GetInfo()
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
	playBar.Hide()

	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))
	cardBackground.CornerRadius = theme.Padding() * 2
	cardBackground.StrokeWidth = 2
	cardBackground.StrokeColor = theme.Color(theme.ColorNameSeparator)

	thumbnailMask := canvas.NewRectangle(color.Transparent)
	thumbnailMask.CornerRadius = theme.Padding() * 1.5
	thumbnailMask.StrokeWidth = theme.Padding() / 2
	thumbnailMask.StrokeColor = theme.Color(theme.ColorNameHeaderBackground)

	go ig.RenderLoop()

	return &ItemRenderer{
		ig: ig,

		Background:    cardBackground,
		ThumbnailMask: thumbnailMask,
		InfoLeft:      container.NewVBox(group, adder, statusText),
		InfoRight:     container.NewVBox(id, progress, sizeText),
		InfoBottom:    container.NewVBox(errorText, playBar),

		ProgressBar: progress,
		StatusText:  statusText,
		ErrorText:   errorText,
		SizeText:    sizeText,
		PlayBar:     playBar,
		FavoriteBtn: button.NewFavoriteBtn(info.ID, info.Title),
		Thumbnail:   widgets.NewThumbnail(info.ThumbnailURL),
		TitleWidget: title,
	}
}

var playItemMinWidth float32 = 260
var playItemThumbnailShow float32 = 300
var playItemThumbnailStartWidth float32 = 60
var playItemThumbnailMaxWidth float32 = 120
var topHeight float32 = 30

type ItemRenderer struct {
	ig *ItemGui

	Background    *canvas.Rectangle
	ThumbnailMask *canvas.Rectangle
	InfoLeft      fyne.CanvasObject
	InfoRight     fyne.CanvasObject
	InfoBottom    fyne.CanvasObject

	ProgressBar *widgets.SizeProgressBar
	ErrorText   *canvas.Text
	StatusText  *canvas.Text
	SizeText    *canvas.Text
	PlayBar     *widgets.PlayBar
	FavoriteBtn *button.FavoriteBtn
	TitleWidget *widgets.EllipseText
	Thumbnail   *widgets.Thumbnail
}

func (r *ItemRenderer) MinSize() fyne.Size {
	p := theme.Padding()
	totalHeight := topHeight + r.InfoLeft.MinSize().Height + r.InfoBottom.MinSize().Height + p
	return fyne.NewSize(playItemMinWidth, totalHeight)
}

func (r *ItemRenderer) Layout(size fyne.Size) {
	r.Background.Resize(size)
	r.Background.Move(fyne.NewPos(0, 0))
	p := theme.Padding()

	// Top layout: title and favorite button
	btnSize := r.FavoriteBtn.MinSize().Height
	btnP := (topHeight - btnSize) / 2
	favoriteX := size.Width - btnSize - btnP
	r.FavoriteBtn.Resize(fyne.NewSize(btnSize, btnSize))
	r.FavoriteBtn.Move(fyne.NewPos(favoriteX, btnP))

	titleWidth := favoriteX - p - btnP
	titleHeight := r.TitleWidget.MinSize().Height
	r.TitleWidget.Resize(fyne.NewSize(titleWidth, titleHeight))
	r.TitleWidget.Move(fyne.NewPos(p, (topHeight-titleHeight)/2))

	// Bottom layout: play bar or error message
	bottomHeight := r.InfoBottom.MinSize().Height
	bottomY := size.Height - bottomHeight - p
	r.InfoBottom.Resize(fyne.NewSize(size.Width-p*2, bottomHeight))
	r.InfoBottom.Move(fyne.NewPos(p, bottomY))

	// Center layout
	centerY := topHeight
	centerHeight := bottomY - centerY

	// right layout: id, progress, size
	rightInfoX := size.Width - r.InfoRight.MinSize().Width - p
	r.InfoRight.Resize(fyne.NewSize(r.InfoRight.MinSize().Width, centerHeight))
	r.InfoRight.Move(fyne.NewPos(rightInfoX, centerY))

	// left layout: thumbnail(if possible), group, adder, status
	leftInfoX := p
	if size.Width > playItemThumbnailShow {
		thumbnailWidth := playItemThumbnailStartWidth + size.Width - playItemThumbnailShow
		if thumbnailWidth > playItemThumbnailMaxWidth {
			thumbnailWidth = playItemThumbnailMaxWidth
		}

		r.Thumbnail.Resize(fyne.NewSize(thumbnailWidth, centerHeight))
		r.Thumbnail.Move(fyne.NewPos(leftInfoX, centerY))
		r.ThumbnailMask.Resize(fyne.NewSize(thumbnailWidth+p, centerHeight+p))
		r.ThumbnailMask.Move(fyne.NewPos(leftInfoX-p/2, centerY-p/2))

		leftInfoX += thumbnailWidth + p

		r.Thumbnail.Show()
		r.ThumbnailMask.Show()
	} else {
		r.Thumbnail.Hide()
		r.ThumbnailMask.Hide()
	}

	r.InfoLeft.Resize(fyne.NewSize(rightInfoX-leftInfoX, centerHeight))
	r.InfoLeft.Move(fyne.NewPos(leftInfoX, centerY))
}

func (r *ItemRenderer) Refresh() {
	if r.ig.statusChanged {
		r.ig.statusChanged = false
		status := r.ig.ps.GetStatusInfo()

		r.StatusText.Text = status.Status
		r.StatusText.Color = theme.Color(status.Color)

		if status.PreloadError != nil {
			r.ErrorText.Text = status.PreloadError.Error()
			r.ErrorText.Show()
		} else {
			r.ErrorText.Hide()
		}
	}
	if r.ig.progressChanged {
		r.ig.progressChanged = false
		progress := r.ig.ps.GetProgressInfo()

		r.ProgressBar.SetTotalSize(progress.Total)
		r.ProgressBar.SetCurrentSize(progress.Downloaded)

		if progress.Total > 0 {
			r.SizeText.Text = utils.PrettyByteSize(progress.Total)
		} else {
			r.SizeText.Text = i18n.T("placeholder_unknown_size")
		}

		if progress.IsDownloading {
			r.ProgressBar.Show()
			r.SizeText.Hide()
		} else {
			r.ProgressBar.Hide()
			r.SizeText.Show()
		}
	}
	if r.ig.timeChanged {
		r.ig.timeChanged = false
		timeInfo := r.ig.ps.GetTimeInfo()

		if timeInfo.IsPlaying {
			r.PlayBar.Progress = float32(timeInfo.Progress)
			r.PlayBar.Text = timeInfo.Text

			r.PlayBar.Refresh()
			if !r.PlayBar.Visible() {
				r.PlayBar.Show()
				canvas.NewColorRGBAAnimation(
					theme.Color(theme.ColorNameSeparator),
					theme.Color(theme.ColorNamePrimary),
					500*time.Millisecond,
					func(c color.Color) {
						r.Background.StrokeColor = c
					},
				).Start()
				r.ig.listItem.NotifyUpdateMinSize()
			}
		} else {
			if r.PlayBar.Visible() {
				r.PlayBar.Hide()
				canvas.NewColorRGBAAnimation(
					theme.Color(theme.ColorNamePrimary),
					theme.Color(theme.ColorNameSeparator),
					500*time.Millisecond,
					func(c color.Color) {
						r.Background.StrokeColor = c
					},
				).Start()
				r.ig.listItem.NotifyUpdateMinSize()
			}
		}
	}

	canvas.Refresh(r.ig)
}

func (r *ItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Background,
		r.Thumbnail,
		r.ThumbnailMask,

		r.TitleWidget,
		r.FavoriteBtn,

		r.InfoBottom,

		r.InfoLeft,
		r.InfoRight,
	}
}

func (r *ItemRenderer) Destroy() {
	close(r.ig.StopCh)
}
