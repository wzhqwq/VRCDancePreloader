package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

var skipTest = false

func SetSkipTest(b bool) {
	skipTest = b
}

type ProxyTester struct {
	input.Tester

	Status  input.Status
	Message string
	Input   *input.InputWithTester

	Value string
	Item  string
}

func NewProxyTester(item, value string) *ProxyTester {
	return &ProxyTester{
		Status: input.StatusUnknown,
		Value:  value,
		Item:   item,
	}
}

func (t *ProxyTester) Test() {
	if t.Status == input.StatusTesting {
		return
	}
	t.Status = input.StatusTesting
	if t.Input != nil {
		t.Input.SetTestBtn(true)
	}
	go func() {
		ok, message := config.Proxy.Test(t.Item)
		if ok {
			t.Message = i18n.T("tip_connectivity_test_pass")
			t.Status = input.StatusOk
		} else {
			t.Message = message
			t.Status = input.StatusError
		}
		if t.Input != nil {
			t.Input.SetTestBtn(false)
		}
	}()
}

func (t *ProxyTester) Save(value string) {
	t.Value = value
	t.Status = input.StatusUnknown
	config.Proxy.Update(t.Item, value)
	if t.Input != nil {
		t.Input.SetTestBtn(false)
	}
}

func (t *ProxyTester) GetStatus() input.Status {
	return t.Status
}

func (t *ProxyTester) GetValue() string {
	return t.Value
}

func (t *ProxyTester) GetMessage() string {
	return t.Message
}

func (t *ProxyTester) GetInput(label string) *input.InputWithTester {
	if t.Input == nil {
		t.Input = input.NewInputWithTester(t, label)
	}
	return t.Input
}

func (t *ProxyTester) TestIfNotOk() {
	if t.Status == input.StatusOk {
		return
	}
	t.Test()
}
