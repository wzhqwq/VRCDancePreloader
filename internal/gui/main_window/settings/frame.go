package settings

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/host"
)

func CreateSettingsContainer() fyne.CanvasObject {
	cfg := host.Config()
	scroll := container.NewVScroll(
		containers.NewResponsiveFrame(
			3,
			300,
			widgets.NewCard(createHijackSettingsContent(cfg)),
			widgets.NewCard(createProxySettingsContent(cfg)),
			widgets.NewCard(createPreloadSettingsContent(cfg)),
			widgets.NewCard(createThirdPartySettingsContent(cfg)),
			widgets.NewCard(createExecutableSettingsContent(cfg)),
			widgets.NewCard(createDownloadSettingsContent(cfg)),
			widgets.NewCard(createCacheSettingsContent(cfg)),
		),
	)
	scroll.SetMinSize(fyne.NewSize(300, 300))
	scroll.Refresh()

	background := canvas.NewRectangle(theme.Color(custom_fyne.ColorNameOuterBackground))
	c := container.NewStack(background, scroll)

	return c
}
