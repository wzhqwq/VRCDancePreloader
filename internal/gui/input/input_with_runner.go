package input

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type InputWithRunner struct {
	InputWithSave

	service interactive.StatefulService

	StatusIcon *icons.IconWithMessage
}

func NewInputWithRunner[T AcceptedValue](service interactive.StatefulService, setting interactive.StatefulSetting[T], label string) *InputWithRunner {
	i := &InputWithRunner{
		InputWithSave: InputWithSave{},
		service:       service,

		StatusIcon: icons.NewIconWithMessage(nil),
	}

	i.InputAppendItems = []fyne.CanvasObject{container.NewPadded(i.StatusIcon)}

	i.Extend(NewAdapterSetting(setting), label)
	i.AddLifeCycleFn(i.loop)

	i.ExtendBaseWidget(i)

	return i
}

func (i *InputWithRunner) loop(stopCh <-chan struct{}) {
	ch := i.service.SubscribeStatus()
	defer ch.Close()
	i.updateStatus(i.service.Status())

	for {
		select {
		case <-stopCh:
			return
		case status := <-ch.Channel:
			i.updateStatus(status)
		}
	}
}

func (i *InputWithRunner) updateStatus(status interactive.RunnerStatus) {
	// The error comes first: a service that is up can still report a failed self
	// check, and showing a green tick for it would hide the problem the runner
	// just detected.
	if status.Error != nil {
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.WarningIcon(), theme.ColorNameError))
		i.StatusIcon.SetMessage(status.Error.Error(), theme.Color(theme.ColorNameError))
		return
	}

	if status.Running {
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.MediaPlayIcon(), theme.ColorNameSuccess))
		i.StatusIcon.SetMessage("", theme.Color(theme.ColorNameSuccess))
		return
	}

	i.StatusIcon.SetIcon(nil)
	i.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
}
