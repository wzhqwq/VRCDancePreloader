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

	OnClick func()

	active  bool
	hovered bool

	date time.Time
}

func NewRecordButton(date time.Time, active bool) *RecordButton {
	b := &RecordButton{
		date:   date,
		active: active,
	}

	b.ExtendBaseWidget(b)

	return b
}

func (b *RecordButton) SetActive(active bool) {
	b.active = active
	fyne.Do(func() {
		b.Refresh()
	})
}

func (b *RecordButton) CreateRenderer() fyne.WidgetRenderer {
	dateText := canvas.NewText(b.date.Format("01/02"), theme.Color(theme.ColorNameForeground))
	dateText.TextSize = 14
	dateText.TextStyle = fyne.TextStyle{Bold: true}

	timeText := canvas.NewText(b.date.Format("15:04"), theme.Color(theme.ColorNameForeground))
	timeText.TextSize = 12
	timeText.TextStyle = fyne.TextStyle{Italic: true}

	var yearText *canvas.Text
	if b.date.Year() != time.Now().Year() {
		yearText = canvas.NewText(b.date.Format("2006"), theme.Color(theme.ColorNameForeground))
		yearText.TextSize = 12
	}

	background := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	background.CornerRadius = 8
	background.StrokeWidth = 2

	return &RecordButtonRenderer{
		b: b,

		DateText: dateText,
		TimeText: timeText,
		YearText: yearText,

		Background: background,
	}
}

func (b *RecordButton) Tapped(_ *fyne.PointEvent) {
	if b.OnClick != nil {
		b.OnClick()
	}
}

func (b *RecordButton) MouseIn(_ *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}
func (b *RecordButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}
func (b *RecordButton) MouseMoved(_ *desktop.MouseEvent) {
}

type RecordButtonRenderer struct {
	b *RecordButton

	YearText *canvas.Text
	DateText *canvas.Text
	TimeText *canvas.Text

	Background *canvas.Rectangle
}

func (r *RecordButtonRenderer) MinSize() fyne.Size {
	return fyne.NewSize(50, 50)
}

func (r *RecordButtonRenderer) Layout(size fyne.Size) {
	p := float32(2)

	r.Background.Resize(size)
	r.Background.Move(fyne.NewPos(0, 0))

	dateSize := r.DateText.MinSize()
	r.DateText.Resize(dateSize)
	r.DateText.Move(fyne.NewPos((size.Width-dateSize.Width)/2, (size.Height-dateSize.Height)/2))

	timeSize := r.TimeText.MinSize()
	r.TimeText.Resize(timeSize)
	r.TimeText.Move(fyne.NewPos((size.Width-timeSize.Width)/2, size.Height-timeSize.Height-p))

	if r.YearText != nil {
		yearSize := r.YearText.MinSize()
		r.YearText.Resize(yearSize)
		r.YearText.Move(fyne.NewPos((size.Width-yearSize.Width)/2, p))
	}
}

func (r *RecordButtonRenderer) Refresh() {
	if r.b.hovered {
		r.Background.FillColor = theme.Color(theme.ColorNameHover)
	} else {
		r.Background.FillColor = theme.Color(theme.ColorNameButton)
	}
	if r.b.active {
		r.Background.StrokeColor = theme.Color(theme.ColorNamePrimary)
	} else {
		r.Background.StrokeColor = theme.Color(theme.ColorNameSeparator)
	}
	canvas.Refresh(r.b)
}

func (r *RecordButtonRenderer) Objects() []fyne.CanvasObject {
	if r.YearText != nil {
		return []fyne.CanvasObject{
			r.Background,
			r.DateText,
			r.TimeText,
			r.YearText,
		}
	}
	return []fyne.CanvasObject{
		r.Background,
		r.DateText,
		r.TimeText,
	}
}

func (r *RecordButtonRenderer) Destroy() {
}
