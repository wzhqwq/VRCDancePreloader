package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"time"
)

type RecordButton struct {
	widget.BaseWidget
	fyne.Tappable
	desktop.Hoverable

	YearText *canvas.Text
	DateText *canvas.Text
	TimeText *canvas.Text

	Background *canvas.Rectangle

	OnClick func()

	Active bool
}

func NewRecordButton(date time.Time, active bool) *RecordButton {

	dateText := canvas.NewText(date.Format("01/02"), theme.Color(theme.ColorNameForeground))
	dateText.TextSize = 14
	dateText.TextStyle = fyne.TextStyle{Bold: true}

	timeText := canvas.NewText(date.Format("15:04"), theme.Color(theme.ColorNameForeground))
	timeText.TextSize = 12
	timeText.TextStyle = fyne.TextStyle{Italic: true}

	background := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	background.CornerRadius = 8
	background.StrokeWidth = 2

	b := &RecordButton{
		DateText: dateText,
		TimeText: timeText,

		Background: background,
	}

	if date.Year() != time.Now().Year() {
		yearText := canvas.NewText(date.Format("2006"), theme.Color(theme.ColorNameForeground))
		yearText.TextSize = 12
		b.YearText = yearText
	}

	b.ExtendBaseWidget(b)

	b.SetActive(active)

	return b
}

func (b *RecordButton) SetActive(active bool) {
	b.Active = active
	fyne.Do(func() {
		if active {
			b.Background.StrokeColor = theme.Color(theme.ColorNamePrimary)
		} else {
			b.Background.StrokeColor = theme.Color(theme.ColorNameSeparator)
		}
		b.Refresh()
	})
}

func (b *RecordButton) CreateRenderer() fyne.WidgetRenderer {
	return &RecordButtonRenderer{
		b: b,
	}
}

func (b *RecordButton) Tapped(_ *fyne.PointEvent) {
	if b.OnClick != nil {
		b.OnClick()
	}
}

func (b *RecordButton) MouseIn(_ *desktop.MouseEvent) {
	b.Background.FillColor = theme.Color(theme.ColorNameHover)
	b.Refresh()
}
func (b *RecordButton) MouseOut() {
	b.Background.FillColor = theme.Color(theme.ColorNameButton)
	b.Refresh()
}
func (b *RecordButton) MouseMoved(_ *desktop.MouseEvent) {
}

type RecordButtonRenderer struct {
	b *RecordButton
}

func (r *RecordButtonRenderer) MinSize() fyne.Size {
	return fyne.NewSize(50, 50)
}

func (r *RecordButtonRenderer) Layout(size fyne.Size) {
	p := float32(2)

	r.b.Background.Resize(size)
	r.b.Background.Move(fyne.NewPos(0, 0))

	dateSize := r.b.DateText.MinSize()
	r.b.DateText.Resize(dateSize)
	r.b.DateText.Move(fyne.NewPos((size.Width-dateSize.Width)/2, (size.Height-dateSize.Height)/2))

	timeSize := r.b.TimeText.MinSize()
	r.b.TimeText.Resize(timeSize)
	r.b.TimeText.Move(fyne.NewPos((size.Width-timeSize.Width)/2, size.Height-timeSize.Height-p))

	if r.b.YearText != nil {
		yearSize := r.b.YearText.MinSize()
		r.b.YearText.Resize(yearSize)
		r.b.YearText.Move(fyne.NewPos((size.Width-yearSize.Width)/2, p))
	}
}

func (r *RecordButtonRenderer) Refresh() {
	r.b.Background.Refresh()
}

func (r *RecordButtonRenderer) Objects() []fyne.CanvasObject {
	if r.b.YearText != nil {
		return []fyne.CanvasObject{
			r.b.Background,
			r.b.DateText,
			r.b.TimeText,
			r.b.YearText,
		}
	}
	return []fyne.CanvasObject{
		r.b.Background,
		r.b.DateText,
		r.b.TimeText,
	}
}

func (r *RecordButtonRenderer) Destroy() {
}
