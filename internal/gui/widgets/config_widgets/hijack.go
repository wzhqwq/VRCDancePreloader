package config_widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type MultiSelectSites struct {
	widget.BaseWidget

	setting interactive.StatefulSetting[[]string]

	PyPySelected    []string
	WannaSelected   []string
	DuDuSelected    []string
	BiliSelected    []string
	YouTubeSelected []string
}

func NewMultiSelectSites(setting interactive.StatefulSetting[[]string]) *MultiSelectSites {
	selected := setting.Get()

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
	youTubeSelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsYouTubeSite(site)
	})

	m := &MultiSelectSites{
		setting: setting,

		PyPySelected:    pypySelected,
		WannaSelected:   wannaSelected,
		DuDuSelected:    duduSelected,
		BiliSelected:    biliSelected,
		YouTubeSelected: youTubeSelected,
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
	youTubeSelect := widgets.NewMultiSelect(constants.AllYouTubeSites(), m.YouTubeSelected)
	youTubeSelect.OnChange = func(sites []string) {
		m.YouTubeSelected = sites
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
		container.NewCenter(widget.NewLabel("YouTube")),
		youTubeSelect,
	)

	return widget.NewSimpleRenderer(container.NewVBox(label, form))
}

func (m *MultiSelectSites) update() {
	allSites := append(m.PyPySelected, m.WannaSelected...)
	allSites = append(allSites, m.DuDuSelected...)
	allSites = append(allSites, m.BiliSelected...)
	allSites = append(allSites, m.YouTubeSelected...)

	// TODO prevent or revert changes if error occurs
	err := m.setting.Save(allSites)
	if err != nil {
		// pop up
		dialog.NewError(err, custom_fyne.GetParent()).Show()
		return
	}
}

type MultiSelectSitesRenderer struct {
	m *MultiSelectSites

	c *fyne.Container
}
