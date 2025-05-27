package config

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"image/color"
)

var skipTest = false

func SetSkipTest(b bool) {
	skipTest = b
}

type ProxyStatus string

const (
	ProxyStatusUnknown ProxyStatus = "unknown"
	ProxyStatusOk      ProxyStatus = "ok"
	ProxyStatusError   ProxyStatus = "error"
	ProxyStatusTesting ProxyStatus = "testing"
)

type ProxyController struct {
	Status  ProxyStatus
	Message string
	Input   *ProxyInput

	Value string
	Item  string
}

func NewProxyController(item, value string) *ProxyController {
	return &ProxyController{
		Status: ProxyStatusUnknown,
		Value:  value,
		Item:   item,
	}
}

func (c *ProxyController) Test() {
	if c.Status == ProxyStatusTesting {
		return
	}
	c.Status = ProxyStatusTesting
	if c.Input != nil {
		c.Input.SetTestBtn(true)
	}
	go func() {
		ok, message := config.Proxy.Test(c.Item)
		if ok {
			c.Message = i18n.T("tip_connectivity_test_pass")
			c.Status = ProxyStatusOk
		} else {
			c.Message = message
			c.Status = ProxyStatusError
		}
		if c.Input != nil {
			c.Input.SetTestBtn(false)
		}
	}()
}

func (c *ProxyController) Save(value string) {
	c.Value = value
	c.Status = ProxyStatusUnknown
	config.Proxy.Update(c.Item, value)
	if c.Input != nil {
		c.Input.SetTestBtn(false)
	}
}

type ProxyInput struct {
	widgets.InputWithSave

	Controller *ProxyController

	TestBtn    *widget.Button
	StatusIcon *IconWithMessage
}

func NewProxyInput(controller *ProxyController, label string) *ProxyInput {
	t := &ProxyInput{
		InputWithSave: widgets.InputWithSave{},
		Controller:    controller,
	}

	t.StatusIcon = NewIconWithMessage(nil)
	t.InputAppendItems = []fyne.CanvasObject{container.NewPadded(t.StatusIcon)}

	t.TestBtn = widget.NewButton(i18n.T("btn_test"), func() {
		controller.Test()
	})
	t.AfterSaveItems = []fyne.CanvasObject{t.TestBtn}

	t.updateStatus()

	controller.Input = t
	t.OnSave = func() {
		controller.Save(t.Value)
	}

	t.Extend(controller.Value, label)
	t.ExtendBaseWidget(t)

	return t
}

func (i *ProxyInput) updateStatus() {
	switch i.Controller.Status {
	case ProxyStatusOk:
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.ConfirmIcon(), theme.ColorNameSuccess))
		i.StatusIcon.SetMessage(i.Controller.Message, theme.Color(theme.ColorNameSuccess))
	case ProxyStatusError:
		i.StatusIcon.SetIcon(theme.NewColoredResource(theme.WarningIcon(), theme.ColorNameError))
		i.StatusIcon.SetMessage(i.Controller.Message, theme.Color(theme.ColorNameError))
	case ProxyStatusTesting:
		i.StatusIcon.SetIcon(nil)
		i.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
	default:
		i.StatusIcon.SetIcon(nil)
		i.StatusIcon.SetMessage("", theme.Color(theme.ColorNameForeground))
	}
}

func (i *ProxyInput) SetTestBtn(testing bool) {
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

type IconWithMessage struct {
	widget.BaseWidget
	desktop.Hoverable
	desktop.Cursorable

	Icon    *widget.Icon
	Message *canvas.Text
}

func NewIconWithMessage(icon fyne.Resource) *IconWithMessage {
	t := &IconWithMessage{
		Icon:    widget.NewIcon(icon),
		Message: canvas.NewText("", theme.Color(theme.ColorNameForeground)),
	}
	t.Message.TextSize = 12
	t.Message.Hide()
	t.ExtendBaseWidget(t)
	return t
}

func (i *IconWithMessage) SetIcon(icon fyne.Resource) {
	fyne.Do(func() {
		if icon == nil {
			i.Icon.Hide()
		} else {
			i.Icon.SetResource(icon)
			i.Icon.Show()
		}
	})
}

func (i *IconWithMessage) SetMessage(message string, color color.Color) {
	fyne.Do(func() {
		i.Message.Text = message
		i.Message.Color = color
	})
}

func (i *IconWithMessage) MouseIn(*desktop.MouseEvent) {
	i.Message.Show()
}
func (i *IconWithMessage) MouseOut() {
	i.Message.Hide()
}
func (i *IconWithMessage) MouseMoved(*desktop.MouseEvent) {
}
func (i *IconWithMessage) Cursor() desktop.Cursor {
	return desktop.DefaultCursor
}

func (i *IconWithMessage) CreateRenderer() fyne.WidgetRenderer {
	return &iconWithMessageRenderer{
		i: i,
	}
}

type iconWithMessageRenderer struct {
	i *IconWithMessage
}

func (r *iconWithMessageRenderer) MinSize() fyne.Size {
	return fyne.NewSize(20, 20)
}

func (r *iconWithMessageRenderer) Layout(size fyne.Size) {
	if r.i.Icon.Visible() {
		r.i.Icon.Resize(size)
		r.i.Icon.Move(fyne.NewPos(0, 0))
	}

	messageSize := r.i.Message.MinSize()
	r.i.Message.Resize(messageSize)
	r.i.Message.Move(fyne.NewPos(min(115-messageSize.Width, -(messageSize.Width-size.Width)/2), size.Height+5))
}

func (r *iconWithMessageRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.i.Icon, r.i.Message}
}

func (r *iconWithMessageRenderer) Refresh() {
	canvas.Refresh(r.i)
}

func (r *iconWithMessageRenderer) Destroy() {
}
