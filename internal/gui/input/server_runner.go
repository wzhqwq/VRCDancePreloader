package input

import (
	"errors"
	"strconv"

	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type ServerRunner struct {
	Running     bool
	Port        int
	serverError error

	input *InputWithRunner

	OnSave      func(int)
	StartServer func() error
	StopServer  func()

	em *utils.EventManager[int]
}

func (h *ServerRunner) Save(value string) error {
	port, err := strconv.Atoi(value)
	if err != nil {
		return errors.New(i18n.T("tip_port_malformed"))
	}
	if h.OnSave != nil {
		h.OnSave(port)
	}
	h.em.NotifySubscribers(port)
	return nil
}

func (h *ServerRunner) Run() {
	if h.Running {
		h.StopServer()
	}
	go func() {
		h.Running = true
		h.serverError = nil
		if h.input != nil {
			h.input.UpdateStatus()
		}
		if err := h.StartServer(); err != nil {
			h.serverError = err
			h.Running = false
			if h.input != nil {
				h.input.UpdateStatus()
			}
		}
	}()
}

func (h *ServerRunner) GetStatus() Status {
	if h.serverError != nil {
		return StatusError
	}
	return StatusRunning
}

func (h *ServerRunner) GetValue() string {
	return strconv.Itoa(h.Port)
}

func (h *ServerRunner) GetMessage() string {
	if h.serverError != nil {
		return h.serverError.Error()
	}
	return i18n.T("tip_server_running")
}

func (h *ServerRunner) GetInput(label string) *InputWithRunner {
	if h.input == nil {
		h.input = NewInputWithRunner(h, label)
	}
	return h.input
}

func (h *ServerRunner) SubscribePort() *utils.EventSubscriber[int] {
	return h.em.SubscribeEvent()
}

func NewServerRunner(port int) *ServerRunner {
	return &ServerRunner{
		Running: false,
		Port:    port,

		em: utils.NewEventManager[int](),
	}
}
