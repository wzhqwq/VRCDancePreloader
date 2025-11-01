package button

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
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
	background := canvas.NewRectangle(theme.Color(custom_fyne.ColorNameOuterBackground))
	background.CornerRadius = 8

	return &RecordButtonRenderer{
		b: b,

		DateTime:   newDateTime(b.date),
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

	DateTime   *dateTime
	Background *canvas.Rectangle
}

func (r *RecordButtonRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(50)
}

func (r *RecordButtonRenderer) Layout(size fyne.Size) {
	if r.b.active {
		r.Background.Resize(fyne.NewSize(size.Width+30, size.Height))
		r.DateTime.Move(fyne.NewPos(0, 0))
	} else {
		r.Background.Resize(size)
		r.DateTime.Move(fyne.NewPos(0, 0))
	}
	r.Background.Move(fyne.NewPos(0, 0))
	r.DateTime.Resize(size)
}

func (r *RecordButtonRenderer) Refresh() {
	if r.b.active {
		r.Background.FillColor = theme.Color(theme.ColorNameBackground)
		r.Background.Resize(fyne.NewSize(80, 50))
		r.DateTime.Move(fyne.NewPos(2, 0))
	} else {
		r.Background.FillColor = theme.Color(custom_fyne.ColorNameOuterBackground)
		r.Background.Resize(fyne.NewSquareSize(50))
		r.DateTime.Move(fyne.NewPos(0, 0))
	}
	if r.b.hovered {
		r.Background.FillColor = theme.Color(custom_fyne.ColorNameButtonHover)
	}
	canvas.Refresh(r.b)
}

func (r *RecordButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Background,
		r.DateTime,
	}
}

func (r *RecordButtonRenderer) Destroy() {
}

type dateTime struct {
	widget.BaseWidget

	date time.Time
}

func newDateTime(date time.Time) *dateTime {
	d := &dateTime{
		date: date,
	}
	d.ExtendBaseWidget(d)
	return d
}

func (d *dateTime) CreateRenderer() fyne.WidgetRenderer {
	dateText := canvas.NewText(d.date.Format("01/02"), theme.Color(theme.ColorNameForeground))
	dateText.TextSize = 14
	dateText.TextStyle = fyne.TextStyle{Bold: true}

	timeText := canvas.NewText(d.date.Format("15:04"), theme.Color(theme.ColorNameForeground))
	timeText.TextSize = 12
	timeText.TextStyle = fyne.TextStyle{Italic: true}

	var yearText *canvas.Text
	if d.date.Year() != time.Now().Year() {
		yearText = canvas.NewText(d.date.Format("2006"), theme.Color(theme.ColorNameForeground))
		yearText.TextSize = 12
	}

	return &dateTimeRenderer{
		DateText: dateText,
		TimeText: timeText,
		YearText: yearText,
	}
}

type dateTimeRenderer struct {
	YearText *canvas.Text
	DateText *canvas.Text
	TimeText *canvas.Text
}

func (r *dateTimeRenderer) MinSize() fyne.Size {
	return r.DateText.MinSize()
}

func (r *dateTimeRenderer) Layout(size fyne.Size) {
	p := float32(2)

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

func (r *dateTimeRenderer) Refresh() {
	r.YearText.Refresh()
	r.DateText.Refresh()
	r.TimeText.Refresh()
}

func (r *dateTimeRenderer) Objects() []fyne.CanvasObject {
	if r.YearText != nil {
		return []fyne.CanvasObject{
			r.DateText,
			r.TimeText,
			r.YearText,
		}
	}
	return []fyne.CanvasObject{
		r.DateText,
		r.TimeText,
	}
}

func (r *dateTimeRenderer) Destroy() {}
