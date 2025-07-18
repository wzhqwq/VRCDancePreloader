package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

var skipTest = false

func SetSkipTest(b bool) {
	skipTest = b
}

type ProxyTester struct {
	widgets.Tester

	Status  widgets.Status
	Message string
	Input   *widgets.InputWithTester

	Value string
	Item  string
}

func NewProxyTester(item, value string) *ProxyTester {
	return &ProxyTester{
		Status: widgets.StatusUnknown,
		Value:  value,
		Item:   item,
	}
}

func (t *ProxyTester) Test() {
	if t.Status == widgets.StatusTesting {
		return
	}
	t.Status = widgets.StatusTesting
	if t.Input != nil {
		t.Input.SetTestBtn(true)
	}
	go func() {
		ok, message := config.Proxy.Test(t.Item)
		if ok {
			t.Message = i18n.T("tip_connectivity_test_pass")
			t.Status = widgets.StatusOk
		} else {
			t.Message = message
			t.Status = widgets.StatusError
		}
		if t.Input != nil {
			t.Input.SetTestBtn(false)
		}
	}()
}

func (t *ProxyTester) Save(value string) {
	t.Value = value
	t.Status = widgets.StatusUnknown
	config.Proxy.Update(t.Item, value)
	if t.Input != nil {
		t.Input.SetTestBtn(false)
	}
}

func (t *ProxyTester) GetStatus() widgets.Status {
	return t.Status
}

func (t *ProxyTester) GetValue() string {
	return t.Value
}

func (t *ProxyTester) GetMessage() string {
	return t.Message
}

func (t *ProxyTester) GetInput(label string) *widgets.InputWithTester {
	if t.Input == nil {
		t.Input = widgets.NewInputWithTester(t, label)
	}
	return t.Input
}

func (t *ProxyTester) TestIfNotOk() {
	if t.Status == widgets.StatusOk {
		return
	}
	t.Test()
}
