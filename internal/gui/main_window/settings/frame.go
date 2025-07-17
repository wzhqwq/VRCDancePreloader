package settings

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"image/color"
)

func CreateSettingsContainer() fyne.CanvasObject {
	scroll := container.NewVScroll(
		containers.NewResponsiveFrame(
			3,
			300,
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

	background := canvas.NewRectangle(color.Gray{Y: 240})
	c := container.NewStack(background, scroll)

	return c
}
