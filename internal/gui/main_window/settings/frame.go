package settings

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
)

func CreateSettingsContainer() fyne.CanvasObject {
	scroll := container.NewVScroll(
		containers.NewResponsiveFrame(
			3,
			300,
			widgets.NewCard(createHijackSettingsContent()),
			widgets.NewCard(createProxySettingsContent()),
			widgets.NewCard(createKeySettingsContent()),
			widgets.NewCard(createYoutubeSettingsContent()),
			widgets.NewCard(createPreloadSettingsContent()),
			widgets.NewCard(createDownloadSettingsContent()),
			widgets.NewCard(createCacheSettingsContent()),
		),
	)
	scroll.SetMinSize(fyne.NewSize(300, 300))
	scroll.Refresh()

	background := canvas.NewRectangle(theme.Color(custom_fyne.ColorNameOuterBackground))
	c := container.NewStack(background, scroll)

	return c
}
