package input

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
)

type Runner interface {
	Save(value string) error
	Run()

	GetStatus() Status
	GetValue() string
	GetMessage() string
}

type InputWithRunner struct {
	InputWithSave

	runner Runner

	StatusIcon *icons.IconWithMessage
}

func NewInputWithRunner(runner Runner, label string) *InputWithRunner {
	t := &InputWithRunner{
		InputWithSave: InputWithSave{},
		runner:        runner,

		StatusIcon: icons.NewIconWithMessage(nil),
	}

	t.InputAppendItems = []fyne.CanvasObject{container.NewPadded(t.StatusIcon)}

	t.UpdateStatus()

	t.OnSave = func() error {
		err := runner.Save(t.Value)
		if err != nil {
			return err
		}

		runner.Run()
		return nil
	}

	t.Extend(runner.GetValue(), label)
	t.ExtendBaseWidget(t)

	return t
}

func (i *InputWithRunner) UpdateStatus() {
	switch i.runner.GetStatus() {
	case StatusRunning:
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.MediaPlayIcon(), theme.ColorNameSuccess))
		i.StatusIcon.SetMessage(i.runner.GetMessage(), theme.Color(theme.ColorNameSuccess))
	case StatusError:
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.WarningIcon(), theme.ColorNameError))
		i.StatusIcon.SetMessage(i.runner.GetMessage(), theme.Color(theme.ColorNameError))
	default:
		i.StatusIcon.SetIcon(nil)
		i.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
	}
}
