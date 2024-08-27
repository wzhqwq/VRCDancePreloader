package gui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

type PlayItemGui struct {
	ID       int
	Index    int
	Progress binding.Float

	Card        *fyne.Container
	ProgressBar *widget.ProgressBar
	ErrorText   *canvas.Text
	StatusText  *canvas.Text
	SizeText    *canvas.Text
	PlayBar     *PlayBar

	RunningAnimation *fyne.Animation
}

func NewPlayItemGui(rendered *types.PlayItemRendered) *PlayItemGui {
	// Title
	title := NewEllipseText(rendered.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	// ID
	idText := fmt.Sprintf("#%d", rendered.ID)
	if rendered.ID < 0 {
		idText = i18n.T("placeholder_custom_song")
	}
	id := canvas.NewText(idText, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	// Group
	group := canvas.NewText(rendered.Group, theme.Color(theme.ColorNameForeground))
	group.TextSize = 12

	// Adder
	adder := canvas.NewText(rendered.Adder, theme.Color(theme.ColorNamePlaceHolder))
	adder.TextSize = 12

	// Status
	statusText := canvas.NewText(rendered.Status, theme.Color(rendered.StatusColor))
	statusText.TextSize = 14

	// Progress bar
	bProgress := binding.NewFloat()
	progressBar := widget.NewProgressBarWithData(bProgress)

	// Size
	sizeText := canvas.NewText(rendered.Size, theme.Color(theme.ColorNameForeground))
	sizeText.Alignment = fyne.TextAlignTrailing
	sizeText.TextSize = 12

	// Error message
	errorText := canvas.NewText(rendered.ErrorText, theme.Color(theme.ColorNameError))
	errorText.TextSize = 12

	// Play Bar
	playBar := NewPlayBar()

	// Thumbnail
	thumbnail := NewThumbnail(rendered.OriginalVideoURL)

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
	card.Move(calculatePosition(rendered.Index))

	gui := PlayItemGui{
		ID:    rendered.ID,
		Index: rendered.Index,

		Progress: bProgress,

		Card:        card,
		ProgressBar: progressBar,
		StatusText:  statusText,
		ErrorText:   errorText,
		SizeText:    sizeText,
		PlayBar:     playBar,
	}
	gui.Update(rendered)

	return &gui
}

func (p *PlayItemGui) Update(rendered *types.PlayItemRendered) {
	if p.ID != rendered.ID {
		panic("ID mismatch")
	}

	if p.Index != rendered.Index {
		p.RunningAnimation = canvas.NewPositionAnimation(
			calculatePosition(p.Index),
			calculatePosition(rendered.Index),
			300*time.Millisecond,
			p.Card.Move,
		)
		p.RunningAnimation.Start()
		p.Index = rendered.Index
	}

	p.Progress.Set(rendered.DownloadProgress)
	if p.StatusText.Text != rendered.Status {
		p.StatusText.Text = rendered.Status
		p.StatusText.Color = theme.Color(rendered.StatusColor)
		p.StatusText.Refresh()
	}
	if p.ErrorText.Text != rendered.ErrorText {
		p.ErrorText.Text = rendered.ErrorText
		p.ErrorText.Refresh()
	}
	if p.SizeText.Text != rendered.Size {
		p.SizeText.Text = rendered.Size
		p.SizeText.Refresh()
	}

	if rendered.ErrorText != "" {
		p.ErrorText.Show()
	} else {
		p.ErrorText.Hide()
	}
	if rendered.IsDownloading {
		p.ProgressBar.Show()
	} else {
		p.ProgressBar.Hide()
	}
	if rendered.IsPlaying {
		p.PlayBar.Show()
		p.PlayBar.Progress = float32(rendered.PlayProgress)
		p.PlayBar.Text = rendered.PlayTimeText
		p.PlayBar.Refresh()
	} else {
		p.PlayBar.Hide()
	}
}

func (p *PlayItemGui) SlideIn() {
	canvas.NewPositionAnimation(
		fyne.NewPos(p.Card.Size().Width, p.Card.Position().Y),
		calculatePosition(p.Index),
		300*time.Millisecond,
		func(pos fyne.Position) {
			p.Card.Move(pos)
		},
	).Start()
}
func (p *PlayItemGui) SlideOut() {
	if p.RunningAnimation != nil {
		p.RunningAnimation.Stop()
	}
	canvas.NewPositionAnimation(
		p.Card.Position(),
		fyne.NewPos(-p.Card.Size().Width, p.Card.Position().Y),
		300*time.Millisecond,
		p.Card.Move,
	).Start()
}

func calculatePosition(index int) fyne.Position {
	return fyne.NewPos(0, float32(index)*(playItemHeight+theme.Padding())-theme.Padding())
}
