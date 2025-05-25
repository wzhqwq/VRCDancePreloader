package history

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
)

type HistoryGui struct {
	widget.BaseWidget

	activeId int

	Records []*persistence.DanceRecord

	recordButtonsCached map[int]*widgets.RecordButton

	Left        *fyne.Container
	Right       *fyne.Container
	LeftScroll  fyne.CanvasObject
	RightScroll fyne.CanvasObject

	Separator *widget.Separator

	StopCh        chan struct{}
	recordsChange *utils.StringEventSubscriber
}

func NewHistoryGui() *HistoryGui {
	left := container.NewVBox()
	right := container.NewVBox()

	leftScroll := container.NewVScroll(container.NewPadded(left))
	rightScroll := container.NewVScroll(right)

	g := &HistoryGui{
		activeId: -1,

		Left:  left,
		Right: right,

		LeftScroll:  leftScroll,
		RightScroll: rightScroll,

		Separator: widget.NewSeparator(),

		StopCh:        make(chan struct{}),
		recordsChange: persistence.GetLocalRecords().SubscribeEvent(),
	}

	g.ExtendBaseWidget(g)

	go func() {
		g.RenderLoop()
	}()

	return g
}

func (g *HistoryGui) UpdateRecords() {
	records, err := persistence.GetLocalRecords().GetRecords()
	if err != nil {
		log.Println("Error getting records:", err)
	}
	g.Records = records

	g.Left.RemoveAll()
	for _, record := range records {
		button := widgets.NewRecordButton(record.StartTime, g.activeId == record.ID)
		button.OnClick = func() {
			g.SetActive(record.ID)
		}
		g.Left.Add(button)
	}

	fyne.Do(func() {
		g.Left.Refresh()
	})

	hasActive := false
	for _, record := range records {
		if g.activeId == record.ID {
			hasActive = true
		}
	}
	if !hasActive && len(records) > 0 {
		g.SetActive(records[0].ID)
	}
}

func (g *HistoryGui) SetActive(id int) {
	if g.activeId == id {
		return
	}

	g.activeId = id

	for i, record := range g.Records {
		button := g.Left.Objects[i].(*widgets.RecordButton)
		button.SetActive(record.ID == id)
	}
	r, err := persistence.GetLocalRecords().GetRecord(id)
	if err != nil {
		log.Println("Error getting record:", err)
		return
	}

	g.Right.RemoveAll()
	g.Right.Add(NewRecordGui(r))

	fyne.Do(func() {
		g.Right.Refresh()
	})
}

func (g *HistoryGui) RenderLoop() {
	g.UpdateRecords()

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
	return &HistoryGuiRenderer{
		g: g,
	}
}

type HistoryGuiRenderer struct {
	g *HistoryGui
}

func (r *HistoryGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.g.LeftScroll.MinSize().Width+r.g.RightScroll.MinSize().Width, 300)
}

func (r *HistoryGuiRenderer) Layout(size fyne.Size) {
	separateX := r.g.LeftScroll.MinSize().Width
	r.g.LeftScroll.Resize(fyne.NewSize(separateX, size.Height))
	r.g.LeftScroll.Move(fyne.NewPos(0, 0))
	r.g.RightScroll.Resize(fyne.NewSize(size.Width-separateX, size.Height))
	r.g.RightScroll.Move(fyne.NewPos(separateX, 0))

	r.g.Separator.Resize(fyne.NewSize(1, size.Height))
	r.g.Separator.Move(fyne.NewPos(separateX, 0))
}

func (r *HistoryGuiRenderer) Refresh() {
	r.g.LeftScroll.Refresh()
	r.g.RightScroll.Refresh()
}

func (r *HistoryGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.g.LeftScroll,
		r.g.RightScroll,
		r.g.Separator,
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
