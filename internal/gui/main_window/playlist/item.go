package playlist

import (
	"image/color"

	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"

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

	stopCh chan struct{}

	statusChanged   bool
	timeChanged     bool
	progressChanged bool
	infoChanged     bool
}

func NewItemGui(ps *song.PreloadedSong, dl *containers.DynamicList) *ItemGui {
	ig := &ItemGui{
		ps: ps,
		dl: dl,

		stopCh: make(chan struct{}, 10),

		statusChanged:   true,
		timeChanged:     true,
		progressChanged: true,
	}
	ig.listItem = containers.NewDynamicListItem(ps.ID, dl, ig)

	ig.ExtendBaseWidget(ig)

	return ig
}

func (ig *ItemGui) RenderLoop() {
	ch := ig.ps.SubscribeEvent(false)
	defer ch.Close()

	for {
		select {
		case <-ig.stopCh:
			return
		case event := <-ch.Channel:
			switch event {
			case song.StatusChange:
				ig.statusChanged = true
				switch ig.ps.GetPreloadStatus() {
				case song.Removed:
					ig.dl.RemoveItem(ig.ps.ID, true)
					return
				}
			case song.ProgressChange:
				ig.progressChanged = true
			case song.TimeChange:
				ig.timeChanged = true
			case song.BasicInfoChange:
				ig.infoChanged = true
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
	//title := widgets.NewSongTitle(info.ID, info.Title, theme.Color(theme.ColorNameForeground))
	title := widgets.NewEllipseText(info.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Group
	groupText := canvas.NewText(info.Group, theme.Color(theme.ColorNameForeground))
	groupText.TextSize = 14

	// Adder
	adder := canvas.NewText(info.Adder, theme.Color(theme.ColorNamePlaceHolder))
	adder.TextSize = 14

	// ID
	id := canvas.NewText(info.ID, theme.Color(theme.ColorNamePlaceHolder))
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	progress := ig.ps.GetProgressInfo()

	// Progress
	progressBar := widgets.NewSizeProgressBar(0, 0)
	progressBar.Text.TextSize = 10
	progressBar.SetTotalSize(progress.Total)
	progressBar.SetCurrentSize(progress.Downloaded)

	// Size
	size := i18n.T("placeholder_unknown_size")
	if progress.Total > 0 {
		size = utils.PrettyByteSize(progress.Total)
	}
	sizeText := canvas.NewText(size, theme.Color(theme.ColorNameForeground))
	sizeText.Alignment = fyne.TextAlignTrailing
	sizeText.TextSize = 12

	if progress.IsDownloading {
		progressBar.Show()
		sizeText.Hide()
	} else {
		progressBar.Hide()
		sizeText.Show()
	}

	status := ig.ps.GetStatusInfo()
	// Status
	statusText := canvas.NewText(status.Status, theme.Color(status.Color))
	statusText.TextSize = 16

	// Error message
	errorText := canvas.NewText("", theme.Color(theme.ColorNameError))
	errorText.TextSize = 12
	if status.PreloadError != nil {
		errorText.Text = status.PreloadError.Error()
		errorText.Show()
	} else {
		errorText.Hide()
	}

	// Play bar
	playBar := widgets.NewPlayBar()
	// no need to set up because time will flow
	playBar.Hide()

	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	cardBackground.CornerRadius = theme.Padding() * 2
	cardBackground.StrokeWidth = 2
	cardBackground.StrokeColor = theme.Color(theme.ColorNameBackground)

	thumbnailMask := canvas.NewRectangle(color.Transparent)
	thumbnailMask.CornerRadius = theme.Padding() * 1.5
	thumbnailMask.StrokeWidth = theme.Padding() / 2
	thumbnailMask.StrokeColor = theme.Color(theme.ColorNameBackground)

	go ig.RenderLoop()

	r := &ItemRenderer{
		ig: ig,

		TitleWidget:   title,
		Background:    cardBackground,
		ThumbnailMask: thumbnailMask,
		InfoBottom:    container.NewVBox(errorText, playBar),

		ProgressBar: progressBar,
		StatusText:  statusText,
		ErrorText:   errorText,
		SizeText:    sizeText,
		GroupText:   groupText,
		PlayBar:     playBar,
		Thumbnail:   widgets.NewThumbnailWithID(info.ID),
		InfoLeft:    container.NewVBox(groupText, adder, statusText),
		InfoRight:   container.NewVBox(id, progressBar, sizeText),
		FavoriteBtn: button.NewFavoriteBtn(info.ID, info.Title),
	}

	return r
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
	GroupText   *canvas.Text
	PlayBar     *widgets.PlayBar
	FavoriteBtn *button.FavoriteBtn
	//TitleWidget *widgets.SongTitle
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
	sizeChanged := false

	if r.ig.infoChanged {
		r.ig.infoChanged = false
		r.refreshInfo()
	}
	if r.ig.statusChanged {
		r.ig.statusChanged = false
		sizeChanged = r.refreshStatus()
	}
	if r.ig.progressChanged {
		r.ig.progressChanged = false
		r.refreshProgress()
	}
	if r.ig.timeChanged {
		r.ig.timeChanged = false
		sizeChanged = r.refreshTime()
	}

	canvas.Refresh(r.ig)

	if sizeChanged {
		r.ig.listItem.NotifyUpdateMinSize()
	}
}

func (r *ItemRenderer) refreshStatus() bool {
	status := r.ig.ps.GetStatusInfo()

	r.StatusText.Text = status.Status
	r.StatusText.Color = theme.Color(status.Color)

	if status.PreloadError != nil {
		r.ErrorText.Text = status.PreloadError.Error()
		if r.ErrorText.Hidden {
			r.ErrorText.Show()
			return true
		}
	} else {
		if !r.ErrorText.Hidden {
			r.ErrorText.Hide()
			return true
		}
	}
	return false
}

func (r *ItemRenderer) refreshProgress() {
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

func (r *ItemRenderer) refreshTime() bool {
	timeInfo := r.ig.ps.GetTimeInfo()

	if timeInfo.IsPlaying {
		r.Background.StrokeColor = theme.Color(theme.ColorNamePrimary)
		if timeInfo.Progress < 0 {
			// not synced
			if r.PlayBar.Visible() {
				r.PlayBar.Hide()
				return true
			}
		} else {
			r.PlayBar.Progress = float32(timeInfo.Progress)
			r.PlayBar.Text = timeInfo.Text
			r.PlayBar.Refresh()
			if !r.PlayBar.Visible() {
				r.PlayBar.Show()
				return true
			}
		}
	} else {
		r.Background.StrokeColor = theme.Color(theme.ColorNameBackground)
		if r.PlayBar.Visible() {
			r.PlayBar.Hide()
			return true
		}
	}
	return false
}

func (r *ItemRenderer) refreshInfo() {
	info := r.ig.ps.GetInfo()

	r.TitleWidget.Text = info.Title
	r.TitleWidget.Refresh()

	r.GroupText.Text = info.Group
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
	close(r.ig.stopCh)
}
