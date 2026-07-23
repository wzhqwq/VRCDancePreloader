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
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/host"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type BroadcastButton struct {
	button.PaddedIconBtn

	rich *widget.RichText

	cfg *config.Manager
	svc interactive.StatefulService
}

func NewBroadcastButton() *BroadcastButton {
	cfg := host.Config()
	svc := host.LiveServer()

	rich := widget.NewRichTextFromMarkdown(i18n.T("tip_on_live", goeasyi18n.Options{
		Data: map[string]interface{}{
			"Port": cfg.LiveServerPort(),
		},
	}))
	rich.Wrapping = i18n.GetLangWrapping()

	wholeContent := container.NewVBox(
		input.NewCheck(i18n.T("label_enable_broadcast"), cfg.LiveServerEnabled()),
		interactive_widgets.NewAvailableWhen(
			container.NewVBox(
				input.NewInputWithRunner(svc, cfg.LiveServerPort(), i18n.T("label_broadcast_port")),
				rich,
			),
			cfg.LiveServerEnabled(),
		),
	)

	scroll := container.NewVScroll(container.NewPadded(wholeContent))
	scroll.SetMinSize(fyne.NewSize(250, 300))

	btn := &BroadcastButton{
		rich: rich,

		cfg: cfg,
		svc: svc,
	}
	btn.Extend(nil)

	btn.OnClick = func() {
		openBroadcastModal(scroll)
	}

	btn.AddLifeCycleFn(btn.loop)

	btn.ExtendBaseWidget(btn)

	return btn
}

func (b *BroadcastButton) loop(stopCh <-chan struct{}) {
	portCh := b.cfg.LiveServerPort().Subscribe()
	defer portCh.Close()
	statusCh := b.svc.SubscribeStatus()
	defer statusCh.Close()

	b.SetRich(b.cfg.LiveServerPort().Get())
	b.SetStatus(b.svc.Status())

	for {
		select {
		case <-stopCh:
			return
		case port := <-portCh.Channel:
			b.SetRich(port)
		case status := <-statusCh.Channel:
			b.SetStatus(status)
		}
	}
}

func (b *BroadcastButton) SetRich(port int) {
	b.rich.ParseMarkdown(i18n.T("tip_on_live", goeasyi18n.Options{
		Data: map[string]interface{}{
			"Port": port,
		},
	}))
}

func (b *BroadcastButton) SetStatus(status interactive.RunnerStatus) {
	if status.Running {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("broadcast"), theme.ColorNamePrimary))
	} else if status.Error != nil {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("broadcast"), theme.ColorNameError))
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
