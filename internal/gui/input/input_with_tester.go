package input

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type InputWithTester struct {
	InputWithSave

	tester  interactive.StatefulTester
	setting interactive.StatefulSetting[string]

	TestBtn    *widget.Button
	StatusIcon *icons.IconWithMessage
}

func NewInputWithTester(tester interactive.StatefulTester, setting interactive.StatefulSetting[string], label string) *InputWithTester {
	t := &InputWithTester{
		InputWithSave: InputWithSave{},
		tester:        tester,

		StatusIcon: icons.NewIconWithMessage(nil),
	}

	t.InputAppendItems = []fyne.CanvasObject{container.NewPadded(t.StatusIcon)}

	t.TestBtn = widget.NewButton(i18n.T("btn_test"), func() {
		go tester.Test()
	})
	t.AfterSaveItems = []fyne.CanvasObject{t.TestBtn}

	t.Extend(NewAdapterSetting(setting), label)
	t.AddLifeCycleFn(t.loop)
	t.ExtendBaseWidget(t)

	return t
}

func (t *InputWithTester) loop(stopCh <-chan struct{}) {
	ch := t.tester.SubscribeStatus()
	defer ch.Close()
	t.updateStatus(t.tester.Status())

	for {
		select {
		case <-stopCh:
			return
		case status := <-ch.Channel:
			t.updateStatus(status)
		}
	}
}

func (t *InputWithTester) updateStatus(status interactive.TesterStatus) {
	fyne.Do(func() {
		if status.Running {
			t.StatusIcon.SetIcon(nil)
			t.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
			t.TestBtn.SetText(i18n.T("btn_testing"))
			t.TestBtn.Disable()
			return
		}

		t.TestBtn.SetText(i18n.T("btn_test"))
		t.TestBtn.Enable()

		if status.Stale {
			t.StatusIcon.SetIcon(nil)
			t.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
			return
		}

		if status.Error == nil {
			t.StatusIcon.SetIcon(theme.NewColoredResource(theme.ConfirmIcon(), theme.ColorNameSuccess))
			t.StatusIcon.SetMessage("", theme.Color(theme.ColorNameSuccess))
		} else {
			t.StatusIcon.SetIcon(theme.NewColoredResource(theme.WarningIcon(), theme.ColorNameError))
			t.StatusIcon.SetMessage(status.Error.Error(), theme.Color(theme.ColorNameError))
		}
	})
}
