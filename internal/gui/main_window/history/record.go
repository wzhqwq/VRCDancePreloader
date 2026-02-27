package history

import (
	"weak"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type RecordGui struct {
	widget.BaseWidget

	Record *persistence.DanceRecord

	orderChanged   bool
	commentChanged bool
}

func NewRecordGui(record *persistence.DanceRecord) *RecordGui {
	g := &RecordGui{
		Record: record,
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *RecordGui) UpdateOrders() {
	g.orderChanged = true

	fyne.Do(func() {
		g.Refresh()
	})
}

func (g *RecordGui) RenderLoop(stopCh chan struct{}) {
	ch := g.Record.SubscribeEvent()
	defer ch.Close()

	for {
		select {
		case <-stopCh:
			return
		case <-ch.Channel:
			g.UpdateOrders()
		}
	}
}

func (g *RecordGui) HandleRemove(order persistence.Order) {
	g.Record.RemoveOrder(order.Time)
}

func (g *RecordGui) CreateRenderer() fyne.WidgetRenderer {
	titleText := i18n.T("wrapper_record_title", goeasyi18n.Options{
		Data: i18n.ParseDate(g.Record.StartTime),
	})
	title := canvas.NewText(titleText, theme.Color(theme.ColorNameForeground))
	title.TextSize = 18
	title.TextStyle.Bold = true

	comment := widgets.NewEllipseText(utils.FirstLine(g.Record.Comment), theme.Color(theme.ColorNameForeground))
	comment.TextSize = 14

	deleteBtn := widget.NewButton(i18n.T("btn_delete_record"), func() {
		ConfirmDeleteRecord(g.Record.ID)
	})
	deleteBtn.Importance = widget.DangerImportance

	editCommentBtn := widget.NewButton(i18n.T("btn_edit_comment"), func() {
		ShowCommentEditor(g.Record.Comment, func(comment string) {
			g.Record.SetComment(comment)
			g.commentChanged = true
			g.Refresh()
		})
	})

	list := container.NewVBox()

	r := &RecordGuiRenderer{
		g: g,

		stopCh: make(chan struct{}),

		Title:   title,
		Comment: comment,

		DeleteBtn:  deleteBtn,
		CommentBtn: editCommentBtn,

		Separator: widget.NewSeparator(),

		List:   list,
		Scroll: container.NewVScroll(list),

		itemMap: make(map[string]weak.Pointer[OrderGui]),
	}

	r.pushOrders()

	go g.RenderLoop(r.stopCh)

	return r
}

type RecordGuiRenderer struct {
	g *RecordGui

	stopCh chan struct{}

	Title   fyne.CanvasObject
	Comment *widgets.EllipseText

	DeleteBtn  *widget.Button
	CommentBtn *widget.Button

	Separator fyne.CanvasObject

	List   *fyne.Container
	Scroll *container.Scroll

	itemMap map[string]weak.Pointer[OrderGui]
}

var recordTopHeight float32 = 100
var recordMinHeight float32 = 300

func (r *RecordGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(max(r.Title.MinSize().Width, r.List.MinSize().Width), recordMinHeight)
}

func (r *RecordGuiRenderer) Layout(size fyne.Size) {
	p := theme.Padding()

	r.Title.Resize(r.Title.MinSize())
	r.Title.Move(fyne.NewPos(p, p))

	r.Comment.Resize(fyne.NewSize(size.Width-p*2, r.Comment.MinSize().Height))
	r.Comment.Move(fyne.NewPos(p, p*2+r.Title.MinSize().Height))

	actionsLeft := size.Width - r.DeleteBtn.MinSize().Width - p
	actionsTop := recordTopHeight - r.DeleteBtn.MinSize().Height - p

	r.DeleteBtn.Resize(r.DeleteBtn.MinSize())
	r.DeleteBtn.Move(fyne.NewPos(actionsLeft, actionsTop))

	actionsLeft -= r.CommentBtn.MinSize().Width + p
	r.CommentBtn.Resize(r.CommentBtn.MinSize())
	r.CommentBtn.Move(fyne.NewPos(actionsLeft, actionsTop))

	r.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.Separator.Move(fyne.NewPos(0, recordTopHeight))

	r.Scroll.Resize(fyne.NewSize(size.Width, size.Height-recordTopHeight))
	r.Scroll.Move(fyne.NewPos(0, recordTopHeight))
}

func (r *RecordGuiRenderer) pushOrders() {
	items := lo.Map(r.g.Record.GetOrdersSnapshot(), func(order persistence.Order, _ int) *OrderGui {
		if item, ok := r.itemMap[order.Key()]; ok {
			if v := item.Value(); v != nil {
				return v
			}
		}
		orderGui := NewOrderGui(order)
		orderGui.onRemove = func() {
			r.g.HandleRemove(order)
		}
		r.itemMap[order.Key()] = weak.Make(orderGui)
		return orderGui
	})

	if r.List.Objects != nil {
		r.List.RemoveAll()
	}
	for _, item := range items {
		r.List.Add(item)
	}
	r.List.Refresh()
	r.Scroll.Refresh()
}

func (r *RecordGuiRenderer) Refresh() {
	if r.g.orderChanged {
		r.g.orderChanged = false

		r.pushOrders()
	}
	if r.g.commentChanged {
		r.g.commentChanged = false

		r.Comment.Text = utils.FirstLine(r.g.Record.Comment)
		r.Comment.Refresh()
	}

	canvas.Refresh(r.g)
}

func (r *RecordGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Scroll,
		r.Title,
		r.Comment,
		r.DeleteBtn,
		r.CommentBtn,
		r.Separator,
	}
}

func (r *RecordGuiRenderer) Destroy() {
	close(r.stopCh)
}
