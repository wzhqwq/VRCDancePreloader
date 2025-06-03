package history

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type RecordGui struct {
	widget.BaseWidget

	Record *persistence.DanceRecord

	StopCh       chan struct{}
	ordersChange *utils.StringEventSubscriber
}

func NewRecordGui(record *persistence.DanceRecord) *RecordGui {
	g := &RecordGui{
		StopCh: make(chan struct{}),

		Record: record,

		ordersChange: record.SubscribeEvent(),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *RecordGui) UpdateOrders() {
	fyne.Do(func() {
		g.Refresh()
	})
}

func (g *RecordGui) RenderLoop() {
	for {
		select {
		case <-g.StopCh:
			return
		case <-g.ordersChange.Channel:
			g.UpdateOrders()
		}
	}
}

func (g *RecordGui) CreateRenderer() fyne.WidgetRenderer {
	go g.RenderLoop()

	return &RecordGuiRenderer{
		g: g,

		List: container.NewVBox(),
	}
}

type RecordGuiRenderer struct {
	g *RecordGui

	List *fyne.Container
}

func (r *RecordGuiRenderer) MinSize() fyne.Size {
	return r.List.MinSize()
}

func (r *RecordGuiRenderer) Layout(size fyne.Size) {
	r.List.Resize(size)
}

func (r *RecordGuiRenderer) Refresh() {
	r.List.RemoveAll()
	for _, order := range r.g.Record.Orders {
		orderGui := NewOrderGui(order)
		r.List.Add(orderGui)
	}
	r.List.Refresh()

	canvas.Refresh(r.g)
}

func (r *RecordGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.List,
	}
}

func (r *RecordGuiRenderer) Destroy() {
	close(r.g.StopCh)
	r.g.ordersChange.Close()
}
