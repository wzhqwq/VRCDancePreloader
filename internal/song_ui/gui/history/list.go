package history

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
)

type HistoryGui struct {
	widget.BaseWidget

	activeId int

	Records []*persistence.DanceRecord

	recordButtonsCached map[int]*button.RecordButton

	StopCh        chan struct{}
	recordsChange *utils.StringEventSubscriber

	recordsChanged bool
	activeChanged  bool
}

func NewHistoryGui() *HistoryGui {
	records, err := persistence.GetLocalRecords().GetRecords()
	if err != nil {
		log.Println("Error getting records:", err)
	}

	g := &HistoryGui{
		activeId: -1,

		Records: records,

		StopCh:        make(chan struct{}),
		recordsChange: persistence.GetLocalRecords().SubscribeEvent(),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *HistoryGui) UpdateRecords() {
	records, err := persistence.GetLocalRecords().GetRecords()
	if err != nil {
		log.Println("Error getting records:", err)
	}
	g.Records = records

	g.recordsChanged = true

	fyne.Do(func() {
		g.Refresh()
	})
}

func (g *HistoryGui) SetActive(id int) {
	if g.activeId == id {
		return
	}

	g.activeId = id
	g.activeChanged = true

	fyne.Do(func() {
		g.Refresh()
	})
}

func (g *HistoryGui) RenderLoop() {
	for {
		select {
		case <-g.StopCh:
			return
		case <-g.recordsChange.Channel:
			g.UpdateRecords()
		}
	}
}

func (g *HistoryGui) CreateRenderer() fyne.WidgetRenderer {
	left := container.NewVBox()
	right := container.NewVBox()

	leftScroll := container.NewVScroll(container.NewPadded(left))
	rightScroll := container.NewVScroll(right)

	go g.RenderLoop()

	r := &HistoryGuiRenderer{
		g: g,

		Left:  left,
		Right: right,

		LeftScroll:  leftScroll,
		RightScroll: rightScroll,

		Separator: widget.NewSeparator(),
	}

	r.PushRecordButtons()

	return r
}

type HistoryGuiRenderer struct {
	g *HistoryGui

	Left        *fyne.Container
	Right       *fyne.Container
	LeftScroll  fyne.CanvasObject
	RightScroll fyne.CanvasObject

	Separator *widget.Separator
}

func (r *HistoryGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.LeftScroll.MinSize().Width+r.RightScroll.MinSize().Width, 300)
}

func (r *HistoryGuiRenderer) Layout(size fyne.Size) {
	separateX := r.LeftScroll.MinSize().Width
	r.LeftScroll.Resize(fyne.NewSize(separateX, size.Height))
	r.LeftScroll.Move(fyne.NewPos(0, 0))
	r.RightScroll.Resize(fyne.NewSize(size.Width-separateX, size.Height))
	r.RightScroll.Move(fyne.NewPos(separateX, 0))

	r.Separator.Resize(fyne.NewSize(1, size.Height))
	r.Separator.Move(fyne.NewPos(separateX, 0))
}

func (r *HistoryGuiRenderer) PushRecordButtons() {
	for _, record := range r.g.Records {
		b := button.NewRecordButton(record.StartTime, r.g.activeId == record.ID)
		b.OnClick = func() {
			r.g.SetActive(record.ID)
		}
		r.Left.Add(b)
	}
	r.Left.Refresh()
	r.LeftScroll.Refresh()
}

func (r *HistoryGuiRenderer) Refresh() {
	if r.g.recordsChanged {
		r.g.recordsChanged = false

		r.Left.RemoveAll()
		r.PushRecordButtons()

		hasActive := false
		for _, record := range r.g.Records {
			if r.g.activeId == record.ID {
				hasActive = true
			}
		}
		if !hasActive && len(r.g.Records) > 0 {
			r.g.activeId = r.g.Records[0].ID
		}
		r.g.activeChanged = true
	}

	if r.g.activeChanged {
		r.g.activeChanged = false

		id := r.g.activeId
		for i, record := range r.g.Records {
			b := r.Left.Objects[i].(*button.RecordButton)
			b.SetActive(record.ID == id)
		}

		if id != -1 {
			needUpdateRight := true

			if len(r.Right.Objects) > 0 {
				recordGui := r.Right.Objects[0].(*RecordGui)

				if recordGui.Record.ID == id {
					needUpdateRight = false
				}
			}

			if needUpdateRight {
				record, err := persistence.GetLocalRecords().GetRecord(id)
				if err != nil {
					log.Println("Error getting record:", err)
					return
				}
				r.Right.RemoveAll()
				r.Right.Add(NewRecordGui(record))

				r.RightScroll.Refresh()
			}
		}
	}

	canvas.Refresh(r.g)
}

func (r *HistoryGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.LeftScroll,
		r.RightScroll,
		r.Separator,
	}
}

func (r *HistoryGuiRenderer) Destroy() {
	close(r.g.StopCh)
	r.g.recordsChange.Close()
}

func CheckRecordContinuity(parent fyne.Window) {
	if r := persistence.GetLocalRecords().GetNearestRecord(); r != nil {
		dialog.NewConfirm(
			i18n.T("message_title_continue_record"),
			i18n.T("message_continue_record", goeasyi18n.Options{
				Data: map[string]any{"Time": r.StartTime.Format("15:04:05")},
			}),
			func(confirmed bool) {
				persistence.PrepareHistory(confirmed)
			},
			parent,
		).Show()
	} else {
		persistence.PrepareHistory(false)
	}
}
