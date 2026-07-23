package input

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type AcceptedValue interface {
	int | string | float64
}

type AdapterSetting struct {
	s any
}

func (s *AdapterSetting) Get() string {
	if intSetting, ok := s.s.(interactive.StatefulSetting[int]); ok {
		return strconv.Itoa(intSetting.Get())
	}
	if floatSetting, ok := s.s.(interactive.StatefulSetting[float64]); ok {
		return strconv.FormatFloat(floatSetting.Get(), 'f', -1, 64)
	}
	if stringSetting, ok := s.s.(interactive.StatefulSetting[string]); ok {
		return stringSetting.Get()
	}
	return ""
}

func (s *AdapterSetting) Save(value string) error {
	if intSetting, ok := s.s.(interactive.StatefulSetting[int]); ok {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		return intSetting.Save(intValue)
	}
	if floatSetting, ok := s.s.(interactive.StatefulSetting[float64]); ok {
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		return floatSetting.Save(floatValue)
	}
	if stringSetting, ok := s.s.(interactive.StatefulSetting[string]); ok {
		return stringSetting.Save(value)
	}
	return nil
}

func (s *AdapterSetting) Subscribe() *utils.EventSubscriber[string] {
	if intSetting, ok := s.s.(interactive.StatefulSetting[int]); ok {
		return utils.PipeSubEvent(intSetting.Subscribe(), func(in int) (string, bool) {
			return strconv.Itoa(intSetting.Get()), true
		})
	}
	if floatSetting, ok := s.s.(interactive.StatefulSetting[float64]); ok {
		return utils.PipeSubEvent(floatSetting.Subscribe(), func(in float64) (string, bool) {
			return strconv.FormatFloat(floatSetting.Get(), 'f', -1, 64), true
		})
	}
	if stringSetting, ok := s.s.(interactive.StatefulSetting[string]); ok {
		return stringSetting.Subscribe()
	}
	return nil
}

func (s *AdapterSetting) SubscribeWhether(whether func(string) bool) *utils.EventSubscriber[bool] {
	if intSetting, ok := s.s.(interactive.StatefulSetting[int]); ok {
		return intSetting.SubscribeWhether(func(in int) bool {
			return whether(strconv.Itoa(intSetting.Get()))
		})
	}
	if floatSetting, ok := s.s.(interactive.StatefulSetting[float64]); ok {
		return floatSetting.SubscribeWhether(func(in float64) bool {
			return whether(strconv.FormatFloat(floatSetting.Get(), 'f', -1, 64))
		})
	}
	if stringSetting, ok := s.s.(interactive.StatefulSetting[string]); ok {
		return stringSetting.SubscribeWhether(whether)
	}
	return nil
}

func (s *AdapterSetting) Type() string {
	if _, ok := s.s.(interactive.StatefulSetting[int]); ok {
		return "int"
	}
	if _, ok := s.s.(interactive.StatefulSetting[float64]); ok {
		return "float"
	}
	if _, ok := s.s.(interactive.StatefulSetting[string]); ok {
		return "string"
	}
	return "any"
}

var _ interactive.StatefulSetting[string] = (*AdapterSetting)(nil)

func NewAdapterSetting(s any) *AdapterSetting {
	return &AdapterSetting{
		s: s,
	}
}

type InputWithSave struct {
	interactive_widgets.LifeCycleWidget

	Setting *AdapterSetting

	Value string
	Label string

	Policy InputPolicy

	InputAppendItems []fyne.CanvasObject
	AfterSaveItems   []fyne.CanvasObject

	ForceDigits bool
	Dirty       bool
	processing  bool

	dirtyChanged      bool
	processingChanged bool
}

func NewInputWithSave[T AcceptedValue](setting interactive.StatefulSetting[T], label string) *InputWithSave {
	t := &InputWithSave{}

	t.Extend(NewAdapterSetting(setting), label)

	t.ExtendBaseWidget(t)

	return t
}

func (i *InputWithSave) Extend(setting *AdapterSetting, label string) {
	i.Setting = setting
	i.Label = label
	i.Value = setting.Get()

	valueType := setting.Type()
	switch valueType {
	case "int":
		i.Policy = NewPolicy(IntegerFilter(true), IntValidator(false))
	case "float":
		i.Policy = NewPolicy(FloatFilter(true), FloatValidator(false))
	default:
		i.Policy = nopPolicy()
	}

	i.AddLifeCycleFn(i.loop)
}

func (i *InputWithSave) SetValue(value string) {
	if value == i.Value {
		return
	}

	i.processing = true
	i.processingChanged = true
	fyne.Do(i.Refresh)

	go func() {
		defer func() {
			i.processing = false
			i.processingChanged = true
			fyne.Do(i.Refresh)
		}()

		err := i.validate(value)
		if err != nil {
			// pop up
			dialog.NewError(err, custom_fyne.GetParent()).Show()
			return
		}

		i.Value = value
		i.Dirty = false
		i.dirtyChanged = true
	}()
}

func (i *InputWithSave) validate(value string) error {
	err := i.Policy.Validate(value)
	if err != nil {
		return err
	}

	err = i.Setting.Save(value)
	if err != nil {
		return err
	}

	return nil
}

func (i *InputWithSave) loop(stopCh <-chan struct{}) {
	ch := i.Setting.Subscribe()
	i.updateValue(i.Setting.Get())
	defer ch.Close()

	for {
		select {
		case <-stopCh:
			return
		case value := <-ch.Channel:
			i.updateValue(value)
		}
	}
}

func (i *InputWithSave) updateValue(value string) {
	if i.Value == value {
		return
	}
	i.Value = value

	if i.Dirty {
		i.Dirty = false
		i.dirtyChanged = true
	}

	fyne.Do(i.Refresh)
}

func (i *InputWithSave) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText(i.Label, theme.Color(theme.ColorNamePlaceHolder))
	label.TextSize = 12

	input := widget.NewEntry()

	saveBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		i.SetValue(input.Text)
	})
	saveBtn.Importance = widget.HighImportance
	saveBtn.Hide()

	input.SetText(i.Value)
	input.Wrapping = fyne.TextWrapOff
	input.Scroll = container.ScrollNone
	input.Refresh()
	input.OnChanged = func(s string) {
		_, s = i.Policy.Filter(i.Value, s)
		newDirty := strings.Compare(i.Value, s) != 0
		if newDirty != i.Dirty {
			i.Dirty = newDirty
			i.dirtyChanged = true
			i.Refresh()
		}
	}
	input.OnSubmitted = func(s string) {
		i.SetValue(s)
	}

	r := &inputWithSaveRenderer{
		i: i,

		Label:       label,
		InputWidget: input,
		SaveBtn:     saveBtn,
	}

	r.Created(i)

	return r
}

type inputWithSaveRenderer struct {
	interactive_widgets.BaseLifeCycleRenderer

	i *InputWithSave

	Label       *canvas.Text
	InputWidget *widget.Entry
	SaveBtn     *widget.Button
}

func (r *inputWithSaveRenderer) MinSize() fyne.Size {
	return fyne.NewSize(100, r.Label.MinSize().Height+r.InputWidget.MinSize().Height+theme.Padding())
}

func layoutItems(xOffset, yCenter float32, items []fyne.CanvasObject) float32 {
	p := theme.Padding()
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		xOffset -= item.MinSize().Width
		item.Move(fyne.NewPos(xOffset, yCenter-item.MinSize().Height/2))
		item.Resize(item.MinSize())
		xOffset -= p
	}

	return xOffset
}

func (r *inputWithSaveRenderer) Layout(size fyne.Size) {
	r.layout(size)
}

func (r *inputWithSaveRenderer) layout(size fyne.Size) {
	p := theme.Padding()
	r.Label.Move(fyne.NewPos(0, p/2))

	labelHeight := r.Label.MinSize().Height + p
	inputHeight := size.Height - labelHeight
	buttonY := labelHeight + inputHeight/2
	rightOffset := size.Width

	if r.i.Dirty {
		rightOffset -= r.SaveBtn.MinSize().Width
		r.SaveBtn.Move(fyne.NewPos(rightOffset, buttonY-r.SaveBtn.MinSize().Height/2))
		r.SaveBtn.Resize(r.SaveBtn.MinSize())
		rightOffset -= p
	} else {
		rightOffset = layoutItems(rightOffset, buttonY, r.i.AfterSaveItems)
	}

	layoutItems(rightOffset-p, buttonY, r.i.InputAppendItems)

	r.InputWidget.Move(fyne.NewPos(0, labelHeight))
	r.InputWidget.Resize(fyne.NewSize(rightOffset-p, inputHeight))
}

func (r *inputWithSaveRenderer) Refresh() {
	if r.i.dirtyChanged {
		r.i.dirtyChanged = false
		if r.i.Dirty {
			r.SaveBtn.Show()
			for _, item := range r.i.AfterSaveItems {
				item.Hide()
			}
		} else {
			r.InputWidget.SetText(r.i.Value)
			r.SaveBtn.Hide()
			for _, item := range r.i.AfterSaveItems {
				item.Show()
			}
		}
		r.layout(r.i.Size())
	}

	if r.i.processingChanged {
		r.i.processingChanged = false
		if r.i.processing {
			r.SaveBtn.Disable()
			r.InputWidget.Disable()
		} else {
			r.SaveBtn.Enable()
			r.InputWidget.Enable()
		}
	}

	canvas.Refresh(r.i)
}

func (r *inputWithSaveRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.Label, r.InputWidget, r.SaveBtn}
	objects = append(objects, r.i.InputAppendItems...)
	objects = append(objects, r.i.AfterSaveItems...)
	return objects
}
