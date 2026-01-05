package config

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

type MultiSelectSites struct {
	widget.BaseWidget

	PyPySelected  []string
	WannaSelected []string
	DuDuSelected  []string
	BiliSelected  []string
}

func NewMultiSelectSites(selected []string) *MultiSelectSites {
	pypySelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsPyPySite(site)
	})
	wannaSelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsWannaSite(site)
	})
	duduSelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsDuDuSite(site)
	})
	biliSelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsBiliSite(site)
	})

	m := &MultiSelectSites{
		PyPySelected:  pypySelected,
		WannaSelected: wannaSelected,
		DuDuSelected:  duduSelected,
		BiliSelected:  biliSelected,
	}
	m.ExtendBaseWidget(m)
	return m
}

func (m *MultiSelectSites) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText(i18n.T("label_hijack_intercepted_sites"), theme.Color(theme.ColorNamePlaceHolder))
	label.TextSize = 12

	pypySelect := widgets.NewMultiSelect(constants.AllPyPySites(), m.PyPySelected)
	pypySelect.OnChange = func(sites []string) {
		m.PyPySelected = sites
		m.update()
	}
	wannaSelect := widgets.NewMultiSelect(constants.AllWannaSites(), m.WannaSelected)
	wannaSelect.OnChange = func(sites []string) {
		m.WannaSelected = sites
		m.update()
	}
	duduSelect := widgets.NewMultiSelect(constants.AllDuDuSites(), m.DuDuSelected)
	duduSelect.OnChange = func(sites []string) {
		m.DuDuSelected = sites
		m.update()
	}
	biliSelect := widgets.NewMultiSelect(constants.AllBiliSites(), m.BiliSelected)
	biliSelect.OnChange = func(sites []string) {
		m.BiliSelected = sites
		m.update()
	}

	form := container.New(
		layout.NewFormLayout(),
		container.NewCenter(widget.NewLabel("PyPyDance")),
		pypySelect,
		container.NewCenter(widget.NewLabel("WannaDance")),
		wannaSelect,
		container.NewCenter(widget.NewLabel("DuDuFitDance")),
		duduSelect,
		container.NewCenter(widget.NewLabel("BiliBili")),
		biliSelect,
	)

	return widget.NewSimpleRenderer(container.NewVBox(label, form))
}

func (m *MultiSelectSites) update() {
	allSites := append(m.PyPySelected, m.WannaSelected...)
	allSites = append(allSites, m.DuDuSelected...)
	allSites = append(allSites, m.BiliSelected...)

	config.Hijack.UpdateSites(allSites)
}

type MultiSelectSitesRenderer struct {
	m *MultiSelectSites

	c *fyne.Container
}
