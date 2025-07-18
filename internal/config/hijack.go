package config

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/global_state"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"log"
	"strconv"
)

type HijackServerRunner struct {
	Running     bool
	Port        int
	serverError error

	input *widgets.InputWithRunner
}

func (h *HijackServerRunner) Save(value string) {
	port, err := strconv.Atoi(value)
	if err != nil {
		h.serverError = errors.New(i18n.T("tip_port_malformed"))
	}
	config.Hijack.UpdatePort(port)
}

func (h *HijackServerRunner) Run() {
	if h.Running {
		config.Hijack.Stop()
	}
	go func() {
		h.Running = true
		h.serverError = nil
		if h.input != nil {
			h.input.UpdateStatus()
		}
		if err := config.Hijack.startHijack(); err != nil {
			if global_state.IsInGui() {
				h.serverError = err
				h.Running = false
				if h.input != nil {
					h.input.UpdateStatus()
				}
			} else {
				log.Fatalf("HTTP server error: %v", err)
			}
		}
	}()
}

func (h *HijackServerRunner) GetStatus() widgets.Status {
	if h.serverError != nil {
		return widgets.StatusError
	}
	return widgets.StatusRunning
}

func (h *HijackServerRunner) GetValue() string {
	return strconv.Itoa(h.Port)
}

func (h *HijackServerRunner) GetMessage() string {
	if h.serverError != nil {
		return h.serverError.Error()
	}
	return i18n.T("tip_hijack_server_running")
}

func (h *HijackServerRunner) GetInput(label string) *widgets.InputWithRunner {
	if h.input == nil {
		h.input = widgets.NewInputWithRunner(h, label)
	}
	return h.input
}

func NewHijackServerRunner() *HijackServerRunner {
	return &HijackServerRunner{
		Running: false,
		Port:    config.Hijack.ProxyPort,
	}
}

type MultiSelectSites struct {
	widget.BaseWidget

	PyPySelected  []string
	WannaSelected []string
	BiliSelected  []string
}

func NewMultiSelectSites(selected []string) *MultiSelectSites {
	pypySelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsPyPySite(site)
	})
	wannaSelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsWannaSite(site)
	})
	biliSelected := lo.Filter(selected, func(site string, _ int) bool {
		return constants.IsBiliSite(site)
	})

	m := &MultiSelectSites{
		PyPySelected:  pypySelected,
		WannaSelected: wannaSelected,
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
		container.NewCenter(widget.NewLabel("BiliBili")),
		biliSelect,
	)

	return widget.NewSimpleRenderer(container.NewVBox(label, form))
}

func (m *MultiSelectSites) update() {
	allSites := append(m.PyPySelected, m.WannaSelected...)
	allSites = append(allSites, m.BiliSelected...)

	config.Hijack.UpdateSites(allSites)
}

type MultiSelectSitesRenderer struct {
	m *MultiSelectSites

	c *fyne.Container
}
