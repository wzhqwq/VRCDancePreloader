package widgets

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
)

type Pagination struct {
	widget.BaseWidget

	CurrentPage int
	TotalPage   int

	OnPageChange func(page int)
}

func NewPagination() *Pagination {
	p := &Pagination{
		CurrentPage: 1,
		TotalPage:   1,
	}

	p.ExtendBaseWidget(p)

	return p
}

func (p *Pagination) SetCurrentPage(currentPage int) {
	if p.CurrentPage == currentPage {
		return
	}
	p.CurrentPage = currentPage
	p.Refresh()
}

func (p *Pagination) SetTotalPage(totalPage int) {
	if p.TotalPage == totalPage {
		return
	}
	p.TotalPage = max(1, totalPage)
	p.Refresh()
}

func (p *Pagination) handlePageChange() {
	if p.OnPageChange != nil {
		p.OnPageChange(p.CurrentPage)
	}
	p.Refresh()
}

func (p *Pagination) setPage(page int) {
	p.CurrentPage = page
	p.handlePageChange()
}

func (p *Pagination) CreateRenderer() fyne.WidgetRenderer {
	prevBtn := button.NewPaddedIconBtn(icons.GetIcon("angle-left"))
	prevBtn.OnClick = func() {
		if p.CurrentPage > 1 {
			p.setPage(p.CurrentPage - 1)
		}
	}
	prevBtn.SetPadding(theme.Padding() * 2)
	nextBtn := button.NewPaddedIconBtn(icons.GetIcon("angle-right"))
	nextBtn.OnClick = func() {
		if p.CurrentPage < p.TotalPage {
			p.setPage(p.CurrentPage + 1)
		}
	}
	nextBtn.SetPadding(theme.Padding() * 2)
	firstBtn := button.NewPaddedIconBtn(icons.GetIcon("angles-left"))
	firstBtn.OnClick = func() {
		if p.CurrentPage > 1 {
			p.setPage(1)
		}
	}
	firstBtn.SetPadding(theme.Padding() * 2)
	lastBtn := button.NewPaddedIconBtn(icons.GetIcon("angles-right"))
	lastBtn.OnClick = func() {
		if p.CurrentPage < p.TotalPage {
			p.setPage(p.TotalPage)
		}
	}
	lastBtn.SetPadding(theme.Padding() * 2)

	return &paginationRenderer{
		pagination: p,

		PrevBtn:  prevBtn,
		NextBtn:  nextBtn,
		FirstBtn: firstBtn,
		LastBtn:  lastBtn,

		CurrentPageLabel: widget.NewLabel("0/0"),
	}
}

type paginationRenderer struct {
	pagination *Pagination

	PrevBtn  *button.PaddedIconBtn
	NextBtn  *button.PaddedIconBtn
	FirstBtn *button.PaddedIconBtn
	LastBtn  *button.PaddedIconBtn

	CurrentPageLabel *widget.Label
}

func (r *paginationRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 30)
}

func (r *paginationRenderer) Layout(size fyne.Size) {
	left := float32(0)
	right := size.Width

	r.FirstBtn.Resize(fyne.NewSize(30, size.Height))
	r.FirstBtn.Move(fyne.NewPos(left, 0))
	left += 30 + theme.Padding()

	r.PrevBtn.Resize(fyne.NewSize(30, size.Height))
	r.PrevBtn.Move(fyne.NewPos(left, 0))

	right -= 30
	r.LastBtn.Resize(fyne.NewSize(30, size.Height))
	r.LastBtn.Move(fyne.NewPos(right, 0))

	right -= 30 + theme.Padding()
	r.NextBtn.Resize(fyne.NewSize(30, size.Height))
	r.NextBtn.Move(fyne.NewPos(right, 0))

	labelSize := r.CurrentPageLabel.MinSize()
	r.CurrentPageLabel.Resize(labelSize)
	r.CurrentPageLabel.Move(fyne.NewPos((size.Width-labelSize.Width)/2, (size.Height-labelSize.Height)/2))
}

func (r *paginationRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.PrevBtn,
		r.NextBtn,
		r.FirstBtn,
		r.LastBtn,
		r.CurrentPageLabel,
	}
}

func (r *paginationRenderer) Refresh() {
	if r.pagination.CurrentPage < 1 {
		r.pagination.setPage(1)
	}
	if r.pagination.CurrentPage > r.pagination.TotalPage {
		r.pagination.setPage(r.pagination.CurrentPage)
	}

	r.CurrentPageLabel.SetText(fmt.Sprintf("%d/%d", r.pagination.CurrentPage, r.pagination.TotalPage))

	canvas.Refresh(r.pagination)
}

func (r *paginationRenderer) Destroy() {
}
