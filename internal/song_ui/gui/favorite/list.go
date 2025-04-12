package favorite

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func NewFavoritesPage() fyne.CanvasObject {
	return NewFavoritesGui().Container
}

type FavoritesGui struct {
	Container  *fyne.Container
	List       *fyne.Container
	Pagination *widgets.Pagination

	SortBySelect *widget.Select
	OrderSelect  *widget.Select
	RefreshBtn   *widget.Button

	SortByOptions []string
	OrderOptions  []string

	Favorites *persistence.LocalSongs

	StopCh         chan struct{}
	favoriteChange *utils.StringEventSubscriber
}

const pageSize = 20

func NewFavoritesGui() *FavoritesGui {
	favorites := persistence.GetLocalSongs()
	list := container.NewVBox()
	scroll := container.NewVScroll(container.NewPadded(list))
	scroll.SetMinSize(fyne.NewSize(300, 400))

	sortByOptions := []string{
		i18n.T("option_sort_by_id"),
		i18n.T("option_sort_by_like"),
		i18n.T("option_sort_by_skill"),
	}
	orderOptions := []string{
		i18n.T("option_order_asc"),
		i18n.T("option_order_desc"),
	}

	sortBySelect := widget.NewSelect(sortByOptions, nil)
	sortBySelect.Selected = sortByOptions[0]
	orderSelect := widget.NewSelect(orderOptions, nil)
	orderSelect.Selected = orderOptions[0]
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), nil)

	topBar := container.NewHBox(sortBySelect, orderSelect, refreshBtn)

	pagination := widgets.NewPagination()
	pagination.SetTotalPage(favorites.CalculateTotalPages(pageSize))
	pagination.SetCurrentPage(1)
	total := container.NewBorder(
		container.NewPadded(topBar),
		container.NewPadded(container.NewCenter(pagination)),
		nil,
		nil,
		scroll,
	)

	g := &FavoritesGui{
		Container:  total,
		List:       list,
		Pagination: pagination,

		SortBySelect: sortBySelect,
		OrderSelect:  orderSelect,
		RefreshBtn:   refreshBtn,

		SortByOptions: sortByOptions,
		OrderOptions:  orderOptions,

		Favorites: favorites,

		StopCh:         make(chan struct{}),
		favoriteChange: favorites.SubscribeEvent(),
	}

	refreshFunc := func(string) {
		g.refreshItems()
	}
	sortBySelect.OnChanged = refreshFunc
	orderSelect.OnChanged = refreshFunc
	refreshBtn.OnTapped = func() {
		g.refreshItems()
	}
	pagination.OnPageChange = func(page int) {
		g.refreshItems()
	}

	go g.RenderLoop()

	return g
}

func (fg *FavoritesGui) RenderLoop() {
	fg.refreshItems()

	for {
		select {
		case <-fg.StopCh:
			return
		case <-fg.favoriteChange.Channel:
			fg.refreshItems()
		}
	}
}

func (fg *FavoritesGui) refreshItems() {
	sortBy := "id"
	if fg.SortBySelect.Selected == fg.SortByOptions[1] {
		sortBy = "like"
	}
	if fg.SortBySelect.Selected == fg.SortByOptions[2] {
		sortBy = "skill"
	}
	ascending := fg.OrderSelect.Selected == fg.OrderOptions[0]

	entries := persistence.GetLocalSongs().ListFavorites(fg.Pagination.CurrentPage-1, pageSize, sortBy, ascending)
	fg.List.RemoveAll()
	for _, entry := range entries {
		fg.List.Add(NewItemGui(entry))
	}
	fg.List.Refresh()
	fg.Container.Refresh()
}
