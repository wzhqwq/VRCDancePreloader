package history

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type RecordGui struct {
	widget.BaseWidget

	List   *fyne.Container
	Record *persistence.DanceRecord

	StopCh       chan struct{}
	ordersChange *utils.StringEventSubscriber
}

func NewRecordGui(record *persistence.DanceRecord) *RecordGui {
	list := container.NewVBox()

	g := &RecordGui{
		List:         list,
		Record:       record,
		StopCh:       make(chan struct{}),
		ordersChange: record.SubscribeEvent(),
	}

	g.ExtendBaseWidget(g)

	go func() {
		g.RenderLoop()
	}()

	return g
}

func (g *RecordGui) UpdateOrders() {
	fyne.Do(func() {
		g.List.RemoveAll()

		for _, order := range g.Record.Orders {
			orderGui := NewOrderGui(order)
			g.List.Add(orderGui)
		}
		g.List.Refresh()
	})
}

func (g *RecordGui) RenderLoop() {
	g.UpdateOrders()

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
	return &RecordGuiRenderer{
		g: g,
	}
}

type RecordGuiRenderer struct {
	g *RecordGui
}

func (r *RecordGuiRenderer) MinSize() fyne.Size {
	return r.g.List.MinSize()
}

func (r *RecordGuiRenderer) Layout(size fyne.Size) {
	r.g.List.Resize(size)
}

func (r *RecordGuiRenderer) Refresh() {
	r.g.List.Refresh()
}

func (r *RecordGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.g.List,
	}
}

func (r *RecordGuiRenderer) Destroy() {
	close(r.g.StopCh)
	r.g.ordersChange.Close()
}
