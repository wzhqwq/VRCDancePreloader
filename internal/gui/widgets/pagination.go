package widgets

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
)

type Pagination struct {
	widget.BaseWidget

	CurrentPage int
	TotalPage   int

	PrevBtn  *PaddedIconBtn
	NextBtn  *PaddedIconBtn
	FirstBtn *PaddedIconBtn
	LastBtn  *PaddedIconBtn

	CurrentPageLabel *widget.Label

	OnPageChange func(page int)
}

func NewPagination() *Pagination {
	p := &Pagination{}

	p.PrevBtn = NewPaddedIconBtn(icons.GetIcon("angle-left"))
	p.PrevBtn.OnClick = func() {
		if p.CurrentPage > 1 {
			p.CurrentPage--
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		}
	}
	p.PrevBtn.SetPadding(theme.Padding() * 2)
	p.NextBtn = NewPaddedIconBtn(icons.GetIcon("angle-right"))
	p.NextBtn.OnClick = func() {
		if p.CurrentPage < p.TotalPage {
			p.CurrentPage++
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		}
	}
	p.NextBtn.SetPadding(theme.Padding() * 2)
	p.FirstBtn = NewPaddedIconBtn(icons.GetIcon("angles-left"))
	p.FirstBtn.OnClick = func() {
		if p.CurrentPage > 1 {
			p.CurrentPage = 1
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		}
	}
	p.FirstBtn.SetPadding(theme.Padding() * 2)
	p.LastBtn = NewPaddedIconBtn(icons.GetIcon("angles-right"))
	p.LastBtn.OnClick = func() {
		if p.CurrentPage < p.TotalPage {
			p.CurrentPage = p.TotalPage
			if p.OnPageChange != nil {
				p.OnPageChange(p.CurrentPage)
			}
		}
	}
	p.LastBtn.SetPadding(theme.Padding() * 2)

	p.CurrentPageLabel = widget.NewLabel("0/0")

	p.ExtendBaseWidget(p)

	return p
}

func (p *Pagination) SetCurrentPage(currentPage int) {
	if p.CurrentPage == currentPage {
		return
	}
	p.CurrentPage = currentPage
	fyne.Do(func() {
		p.CurrentPageLabel.SetText(fmt.Sprintf("%d/%d", p.CurrentPage, p.TotalPage))
		p.Refresh()
	})
}

func (p *Pagination) SetTotalPage(totalPage int) {
	if p.TotalPage == totalPage {
		return
	}
	p.TotalPage = totalPage
	fyne.Do(func() {
		p.CurrentPageLabel.SetText(fmt.Sprintf("%d/%d", p.CurrentPage, p.TotalPage))
		p.Refresh()
	})
}

func (p *Pagination) CreateRenderer() fyne.WidgetRenderer {
	return &paginationRenderer{
		pagination: p,
	}
}

type paginationRenderer struct {
	pagination *Pagination
}

func (r *paginationRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 30)
}

func (r *paginationRenderer) Layout(size fyne.Size) {
	p := r.pagination

	left := float32(0)
	right := size.Width

	p.FirstBtn.Resize(fyne.NewSize(30, size.Height))
	p.FirstBtn.Move(fyne.NewPos(left, 0))
	left += 30 + theme.Padding()

	p.PrevBtn.Resize(fyne.NewSize(30, size.Height))
	p.PrevBtn.Move(fyne.NewPos(left, 0))

	right -= 30 + theme.Padding()
	p.LastBtn.Resize(fyne.NewSize(30, size.Height))
	p.LastBtn.Move(fyne.NewPos(right, 0))

	right -= 30 + theme.Padding()
	p.NextBtn.Resize(fyne.NewSize(30, size.Height))
	p.NextBtn.Move(fyne.NewPos(right, 0))

	labelSize := p.CurrentPageLabel.MinSize()
	p.CurrentPageLabel.Resize(labelSize)
	p.CurrentPageLabel.Move(fyne.NewPos((size.Width-labelSize.Width)/2, (size.Height-labelSize.Height)/2))
}

func (r *paginationRenderer) Objects() []fyne.CanvasObject {
	p := r.pagination

	return []fyne.CanvasObject{
		p.PrevBtn,
		p.NextBtn,
		p.FirstBtn,
		p.LastBtn,
		p.CurrentPageLabel,
	}
}

func (r *paginationRenderer) Refresh() {
	r.Layout(r.pagination.Size())
}

func (r *paginationRenderer) Destroy() {
}
