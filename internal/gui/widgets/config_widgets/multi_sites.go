package config_widgets

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type sitesSetting interactive.StatefulSetting[[]string]

type settingSubset struct {
	value    []string
	filterFn func(string) bool
	saveFn   func() error

	em *utils.EventManager[[]string]
}

func (s *settingSubset) Get() []string {
	return s.value
}

func (s *settingSubset) Save(value []string) error {
	s.value = value
	return s.saveFn()
}

func (s *settingSubset) Subscribe() *utils.EventSubscriber[[]string] {
	return s.em.SubscribeEvent()
}

func (s *settingSubset) SubscribeWhether(whether func([]string) bool) *utils.EventSubscriber[bool] {
	return utils.PipeEvent(s.em, func(in []string) (bool, bool) {
		return whether(in), true
	})
}

func (s *settingSubset) setFullSet(full []string) {
	newValue := lo.Filter(full, func(item string, _ int) bool {
		return s.filterFn(item)
	})
	if utils.IsArrayChanged(s.value, newValue) {
		s.value = newValue
		s.em.NotifySubscribers(s.value)
	}
}

var _ sitesSetting = (*settingSubset)(nil)

func newSettingSubset(full []string, filterFn func(string) bool, saveFn func() error) *settingSubset {
	return &settingSubset{
		value: lo.Filter(full, func(item string, _ int) bool {
			return filterFn(item)
		}),
		filterFn: filterFn,
		saveFn:   saveFn,

		em: utils.NewEventManager[[]string](),
	}
}

type MultiSelectSites struct {
	interactive_widgets.LifeCycleWidget

	setting sitesSetting

	pypySetting, wannaSetting, duduSetting, biliSetting, ytSetting *settingSubset
}

func NewMultiSelectSites(setting sitesSetting) *MultiSelectSites {
	selected := setting.Get()

	m := &MultiSelectSites{
		setting: setting,
	}

	saveFn := func() error {
		return setting.Save(slices.Concat(
			m.pypySetting.Get(), m.wannaSetting.Get(), m.duduSetting.Get(), m.biliSetting.Get(), m.ytSetting.Get(),
		))
	}

	m.pypySetting = newSettingSubset(selected, constants.IsPyPySite, saveFn)
	m.wannaSetting = newSettingSubset(selected, constants.IsWannaSite, saveFn)
	m.duduSetting = newSettingSubset(selected, constants.IsDuDuSite, saveFn)
	m.biliSetting = newSettingSubset(selected, constants.IsBiliSite, saveFn)
	m.ytSetting = newSettingSubset(selected, constants.IsYouTubeSite, saveFn)

	m.ExtendBaseWidget(m)
	m.AddLifeCycleFn(m.loop)
	return m
}

func (m *MultiSelectSites) update(value []string) {
	m.pypySetting.setFullSet(value)
	m.wannaSetting.setFullSet(value)
	m.duduSetting.setFullSet(value)
	m.biliSetting.setFullSet(value)
	m.ytSetting.setFullSet(value)
}

func (m *MultiSelectSites) loop(stopCh <-chan struct{}) {
	ch := m.setting.Subscribe()
	defer ch.Close()

	m.update(m.setting.Get())

	for {
		select {
		case <-stopCh:
			return
		case value := <-ch.Channel:
			m.update(value)
		}
	}
}

func (m *MultiSelectSites) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText(i18n.T("label_hijack_intercepted_sites"), theme.Color(theme.ColorNamePlaceHolder))
	label.TextSize = 12

	form := container.New(
		layout.NewFormLayout(),
		container.NewCenter(widget.NewLabel("PyPyDance")),
		widgets.NewMultiSelect(input.NamedOptionFromString(constants.AllPyPySites()), m.pypySetting),
		container.NewCenter(widget.NewLabel("WannaDance")),
		widgets.NewMultiSelect(input.NamedOptionFromString(constants.AllWannaSites()), m.wannaSetting),
		container.NewCenter(widget.NewLabel("DuDuFitDance")),
		widgets.NewMultiSelect(input.NamedOptionFromString(constants.AllDuDuSites()), m.duduSetting),
		container.NewCenter(widget.NewLabel("BiliBili")),
		widgets.NewMultiSelect(input.NamedOptionFromString(constants.AllBiliSites()), m.biliSetting),
		container.NewCenter(widget.NewLabel("YouTube")),
		widgets.NewMultiSelect(input.NamedOptionFromString(constants.AllYouTubeSites()), m.ytSetting),
	)

	r := &multiSelectSitesRenderer{
		c: container.NewVBox(label, form),
	}
	r.objects = []fyne.CanvasObject{r.c}
	r.Created(m)

	return r
}

type multiSelectSitesRenderer struct {
	interactive_widgets.BaseLifeCycleRenderer

	c *fyne.Container

	objects []fyne.CanvasObject
}

func (r *multiSelectSitesRenderer) Layout(size fyne.Size) {
	r.c.Move(fyne.NewPos(0, 0))
	r.c.Resize(size)
}

func (r *multiSelectSitesRenderer) MinSize() fyne.Size {
	return r.c.MinSize()
}

func (r *multiSelectSitesRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *multiSelectSitesRenderer) Refresh() {
	r.c.Refresh()
}
