package button

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

const tipModalMinWidth = 250
const tipModalMinHeight = 300

func NewTipButton(contentKey string) *PaddedIconBtn {
	rich := widget.NewRichTextFromMarkdown(i18n.T(contentKey))
	rich.Wrapping = i18n.GetLangWrapping()
	scroll := container.NewVScroll(rich)
	scroll.SetMinSize(fyne.NewSize(tipModalMinWidth, tipModalMinHeight))

	b := NewPaddedIconBtn(theme.NewColoredResource(theme.Icon(theme.IconNameHelp), theme.ColorNamePlaceHolder))
	b.OnClick = func() {
		openTipModal(scroll)
	}

	return b
}

func openTipModal(content fyne.CanvasObject) {
	dialog.NewCustom(
		i18n.T("message_title_tip"),
		i18n.T("btn_tip_close"),
		content,
		custom_fyne.GetParent(),
	).Show()
}
