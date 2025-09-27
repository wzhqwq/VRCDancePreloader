package input

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

type Tester interface {
	Save(value string)
	Test()

	GetStatus() Status
	GetValue() string
	GetMessage() string
}

type InputWithTester struct {
	InputWithSave

	tester Tester

	TestBtn    *widget.Button
	StatusIcon *icons.IconWithMessage
}

func NewInputWithTester(tester Tester, label string) *InputWithTester {
	t := &InputWithTester{
		InputWithSave: InputWithSave{},
		tester:        tester,

		StatusIcon: icons.NewIconWithMessage(nil),
	}

	t.InputAppendItems = []fyne.CanvasObject{container.NewPadded(t.StatusIcon)}

	t.TestBtn = widget.NewButton(i18n.T("btn_test"), func() {
		tester.Test()
	})
	t.AfterSaveItems = []fyne.CanvasObject{t.TestBtn}

	t.updateStatus()

	t.OnSave = func() {
		tester.Save(t.Value)
	}

	t.Extend(tester.GetValue(), label)
	t.ExtendBaseWidget(t)

	return t
}

func (i *InputWithTester) updateStatus() {
	switch i.tester.GetStatus() {
	case StatusOk:
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.ConfirmIcon(), theme.ColorNameSuccess))
		i.StatusIcon.SetMessage(i.tester.GetMessage(), theme.Color(theme.ColorNameSuccess))
	case StatusError:
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.WarningIcon(), theme.ColorNameError))
		i.StatusIcon.SetMessage(i.tester.GetMessage(), theme.Color(theme.ColorNameError))
	case StatusTesting:
		i.StatusIcon.SetIcon(nil)
		i.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
	default:
		i.StatusIcon.SetIcon(nil)
		i.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
	}
}

func (i *InputWithTester) SetTestBtn(testing bool) {
	i.updateStatus()

	fyne.Do(func() {
		if testing {
			i.TestBtn.SetText(i18n.T("btn_testing"))
			i.TestBtn.Disable()
		} else {
			i.TestBtn.SetText(i18n.T("btn_test"))
			i.TestBtn.Enable()
		}
		i.Refresh()
	})
}
