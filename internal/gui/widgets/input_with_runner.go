package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

type Runner interface {
	Save(value string)
	Run()

	GetStatus() Status
	GetValue() string
	GetMessage() string
}

type InputWithRunner struct {
	InputWithSave

	runner Runner

	StatusIcon *IconWithMessage
}

func NewInputWithRunner(runner Runner, label string) *InputWithRunner {
	t := &InputWithRunner{
		InputWithSave: InputWithSave{},
		runner:        runner,

		StatusIcon: NewIconWithMessage(nil),
	}

	t.InputAppendItems = []fyne.CanvasObject{container.NewPadded(t.StatusIcon)}

	t.UpdateStatus()

	t.OnSave = func() {
		runner.Save(t.Value)
		runner.Run()
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
