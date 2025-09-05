package playlist

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

type BroadcastButton struct {
	button.PaddedIconBtn
}

func NewBroadcastButton() *BroadcastButton {
	liveConfig := config.GetLiveConfig()

	wholeContent := container.NewVBox()

	scroll := container.NewVScroll(container.NewPadded(wholeContent))
	scroll.SetMinSize(fyne.NewSize(250, 300))

	btn := &BroadcastButton{}
	btn.Extend(nil)

	btn.OnClick = func() {
		openBroadcastModal(scroll)
	}

	input := liveConfig.LiveRunner.GetInput(i18n.T("label_broadcast_port"))

	rich := widget.NewRichTextFromMarkdown(i18n.T("tip_on_live", goeasyi18n.Options{
		Data: map[string]interface{}{
			"Port": liveConfig.Port,
		},
	}))
	rich.Wrapping = i18n.GetLangWrapping()

	enableCb := widget.NewCheck(i18n.T("label_enable_broadcast"), func(b bool) {
		if liveConfig.Enabled == b {
			return
		}
		liveConfig.UpdateEnable(b)
		btn.SetLive(b)
		if b {
			input.Show()
			rich.Show()
		} else {
			input.Hide()
			rich.Hide()
		}
	})
	enableCb.Checked = liveConfig.Enabled

	wholeContent.Add(enableCb)
	wholeContent.Add(input)
	wholeContent.Add(rich)

	portChanged := liveConfig.LiveRunner.SubscribePort()
	go func() {
		ch := portChanged.Channel
		for port := range ch {
			rich.ParseMarkdown(i18n.T("tip_on_live", goeasyi18n.Options{
				Data: map[string]interface{}{
					"Port": port,
				},
			}))
		}
	}()
	btn.OnDestroy = func() {
		portChanged.Close()
	}

	btn.ExtendBaseWidget(btn)

	btn.SetLive(liveConfig.Enabled)

	return btn
}

func (b *BroadcastButton) SetLive(live bool) {
	if live {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("broadcast"), theme.ColorNamePrimary))
	} else {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("broadcast"), theme.ColorNamePlaceHolder))
	}
}

func openBroadcastModal(content fyne.CanvasObject) {
	dialog.NewCustom(
		i18n.T("message_title_broadcast"),
		i18n.T("btn_close"),
		content,
		custom_fyne.GetParent(),
	).Show()
}
