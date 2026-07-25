package config_widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type AllowedResources struct {
	widget.BaseWidget

	pypySetting, wannaSetting, duduSetting, biliSetting, ytSetting interactive.StatefulSetting[[]string]
}

func NewAllowedResources(cfg *config.Manager) *AllowedResources {
	m := &AllowedResources{
		pypySetting:  cfg.ThirdPartyPyPyDanceAllowedResources(),
		wannaSetting: cfg.ThirdPartyWannaDanceAllowedResources(),
		duduSetting:  cfg.ThirdPartyDuDuFitDanceAllowedResources(),
		biliSetting:  cfg.ThirdPartyBiliBiliAllowedResources(),
		ytSetting:    cfg.ThirdPartyYouTubeAllowedResources(),
	}

	m.ExtendBaseWidget(m)
	return m
}

func (m *AllowedResources) CreateRenderer() fyne.WidgetRenderer {
	var roomResources = []input.NamedOption[string]{
		{"video", i18n.T("option_videos")},
		{"catalog", i18n.T("option_catalog")},
		{"thumbnail", i18n.T("option_thumbnails")},
	}

	var platformResources = []input.NamedOption[string]{
		{"video", i18n.T("option_videos")},
		{"info", i18n.T("option_info")},
		{"thumbnail", i18n.T("option_thumbnails")},
	}

	label := canvas.NewText(i18n.T("label_allowed_resources"), theme.Color(theme.ColorNamePlaceHolder))
	label.TextSize = 12

	form := container.New(
		layout.NewFormLayout(),
		container.NewCenter(widget.NewLabel("PyPyDance")),
		widgets.NewMultiSelect(roomResources, m.pypySetting),
		container.NewCenter(widget.NewLabel("WannaDance")),
		widgets.NewMultiSelect(roomResources, m.wannaSetting),
		container.NewCenter(widget.NewLabel("DuDuFitDance")),
		widgets.NewMultiSelect(roomResources, m.duduSetting),

		container.NewCenter(widget.NewLabel("BiliBili")),
		widgets.NewMultiSelect(platformResources, m.biliSetting),
		container.NewCenter(widget.NewLabel("YouTube")),
		widgets.NewMultiSelect(platformResources, m.ytSetting),
	)

	return widget.NewSimpleRenderer(container.NewVBox(label, form))
}
