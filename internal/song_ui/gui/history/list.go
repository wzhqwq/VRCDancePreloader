package history

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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
	right := container.NewCenter(widget.NewLabel(i18n.T("tip_select_record")))

	leftScroll := container.NewVScroll(container.NewPadded(left))

	go g.RenderLoop()

	r := &HistoryGuiRenderer{
		g: g,

		Left:       left,
		LeftScroll: leftScroll,

		Right: right,

		Separator: widget.NewSeparator(),
	}

	r.PushRecordButtons()

	return r
}

type HistoryGuiRenderer struct {
	g *HistoryGui

	Left       *fyne.Container
	LeftScroll fyne.CanvasObject

	Right fyne.CanvasObject

	Separator *widget.Separator
}

func (r *HistoryGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.LeftScroll.MinSize().Width+r.Right.MinSize().Width, r.Right.MinSize().Height)
}

func (r *HistoryGuiRenderer) Layout(size fyne.Size) {
	separateX := r.LeftScroll.MinSize().Width
	r.LeftScroll.Resize(fyne.NewSize(separateX, size.Height))
	r.LeftScroll.Move(fyne.NewPos(0, 0))
	r.Right.Resize(fyne.NewSize(size.Width-separateX, size.Height))
	r.Right.Move(fyne.NewPos(separateX, 0))

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

			if gui, ok := r.Right.(*RecordGui); ok {
				if gui.Record.ID == id {
					needUpdateRight = false
				}
			}

			if needUpdateRight {
				record, err := persistence.GetLocalRecords().GetRecord(id)
				if err != nil {
					log.Println("Error getting record:", err)
					return
				}
				r.Right = NewRecordGui(record)
			}
		}
	}

	canvas.Refresh(r.g)
}

func (r *HistoryGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.LeftScroll,
		r.Right,
		r.Separator,
	}
}

func (r *HistoryGuiRenderer) Destroy() {
	close(r.g.StopCh)
	r.g.recordsChange.Close()
}
