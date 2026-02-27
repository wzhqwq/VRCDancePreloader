package favorite

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

type FavoritesGui struct {
	widget.BaseWidget

	Favorites *persistence.LocalSongs

	sortBy     string
	ascending  bool
	page       int
	totalPages int

	SortByOptions []string
	OrderOptions  []string

	entries []*persistence.LocalSongEntry

	active bool
}

const pageSize = 5

func NewFavoritesGui() *FavoritesGui {
	favorites := persistence.GetLocalSongs()

	sortByOptions := []string{
		i18n.T("option_sort_by_id"),
		i18n.T("option_sort_by_like"),
		i18n.T("option_sort_by_skill"),
	}
	orderOptions := []string{
		i18n.T("option_order_asc"),
		i18n.T("option_order_desc"),
	}

	g := &FavoritesGui{
		SortByOptions: sortByOptions,
		OrderOptions:  orderOptions,

		Favorites: favorites,

		sortBy:    "id",
		ascending: true,
	}

	g.ExtendBaseWidget(g)

	return g
}

func (fg *FavoritesGui) refreshItems() {
	fg.entries = persistence.GetLocalSongs().ListFavorites(fg.page, pageSize, fg.sortBy, fg.ascending)
	fg.totalPages = fg.Favorites.CalculateTotalPages(pageSize)
	fg.Refresh()
}

func (fg *FavoritesGui) RenderLoop(stopCh chan struct{}) {
	ch := fg.Favorites.SubscribeEvent()
	defer ch.Close()

	for {
		select {
		case <-stopCh:
			return
		case <-ch.Channel:
			if fg.active {
				fyne.Do(func() {
					fg.refreshItems()
				})
			}
		}
	}
}

func (fg *FavoritesGui) CreateRenderer() fyne.WidgetRenderer {
	list := container.NewVBox()
	scroll := container.NewVScroll(container.NewPadded(list))
	scroll.SetMinSize(fyne.NewSize(300, 400))

	sortBySelect := widget.NewSelect(fg.SortByOptions, func(value string) {
		sortBy := "id"
		if value == fg.SortByOptions[1] {
			sortBy = "like"
		}
		if value == fg.SortByOptions[2] {
			sortBy = "skill"
		}
		fg.sortBy = sortBy

		fg.refreshItems()
	})
	sortBySelect.Selected = fg.SortByOptions[0]

	orderSelect := widget.NewSelect(fg.OrderOptions, func(value string) {
		fg.ascending = value == fg.OrderOptions[0]

		fg.refreshItems()
	})
	orderSelect.Selected = fg.OrderOptions[0]

	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		fg.refreshItems()
	})

	topBar := container.NewHBox(sortBySelect, orderSelect, refreshBtn)

	pagination := widgets.NewPagination()
	pagination.SetTotalPage(fg.totalPages)
	pagination.SetCurrentPage(1)
	pagination.OnPageChange = func(page int) {
		fg.page = page - 1

		fg.refreshItems()
	}

	r := &favoritesGuiRenderer{
		TopBar:     topBar,
		List:       list,
		Scroll:     scroll,
		Pagination: pagination,

		SortBySelect: sortBySelect,
		OrderSelect:  orderSelect,
		RefreshBtn:   refreshBtn,

		g: fg,

		stopCh: make(chan struct{}),
	}

	go fg.RenderLoop(r.stopCh)

	return r
}

func (fg *FavoritesGui) Activate() {
	if fg.active {
		return
	}
	fg.active = true
	fg.refreshItems()
}

type favoritesGuiRenderer struct {
	TopBar     *fyne.Container
	List       *fyne.Container
	Scroll     *container.Scroll
	Pagination *widgets.Pagination

	SortBySelect *widget.Select
	OrderSelect  *widget.Select
	RefreshBtn   *widget.Button

	g *FavoritesGui

	stopCh chan struct{}
}

func (r *favoritesGuiRenderer) MinSize() fyne.Size {
	p := theme.Padding()
	minWidth := r.Scroll.MinSize().Width
	minHeight := r.Scroll.MinSize().Height + r.TopBar.MinSize().Height + r.Pagination.MinSize().Height + p*4
	return fyne.NewSize(minWidth, minHeight)
}

func (r *favoritesGuiRenderer) Layout(s fyne.Size) {
	p := theme.Padding()
	r.TopBar.Move(fyne.NewPos(p, p))
	r.TopBar.Resize(fyne.NewSize(s.Width-p*2, r.TopBar.MinSize().Height))
	offsetTop := r.TopBar.MinSize().Height + p*2

	offsetBottom := s.Height - r.Pagination.MinSize().Height - p
	r.Pagination.Move(fyne.NewPos(p, offsetBottom))
	r.Pagination.Resize(fyne.NewSize(s.Width-p*2, r.Pagination.MinSize().Height))
	offsetBottom -= p

	r.Scroll.Move(fyne.NewPos(0, offsetTop))
	r.Scroll.Resize(fyne.NewSize(s.Width, offsetBottom-offsetTop))
}

func (r *favoritesGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Scroll,
		r.Pagination,
		r.TopBar,
	}
}

func (r *favoritesGuiRenderer) updateItems() {
	if r.List.Objects == nil {
		for _, entry := range r.g.entries {
			r.List.Add(NewItemGui(entry))
		}
		r.List.Refresh()
	} else {
		for i, entry := range r.g.entries {
			if item, ok := r.List.Objects[i].(*ItemGui); ok {
				item.UpdateFavoriteEntry(entry)
			}
		}
	}
}

func (r *favoritesGuiRenderer) Refresh() {
	r.updateItems()
	r.Pagination.SetTotalPage(r.g.totalPages)
}

func (r *favoritesGuiRenderer) Destroy() {
	close(r.stopCh)
}
