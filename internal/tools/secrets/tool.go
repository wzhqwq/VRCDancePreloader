package secrets

import (
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/zalando/go-keyring"
)

const YoutubeKeyUser = "youtube-api-key"
const ObsSecretUser = "obs-websocket"

var cfg Config
var youtubeApiKeyEm *utils.EventManager[string]

type Tool struct {
	service.ConfigurableTool
}

func New(c Config) *Tool {
	cfg = c

	youtubeApiKeyEm = utils.NewEventManager[string]()

	return &Tool{
		ConfigurableTool: service.ConstructStatelessTool(),
	}
}

func Migrate(c Config) error {
	if err := migrateField(YoutubeKeyUser, c.YoutubeApiKey); err != nil {
		return err
	}

	return nil
}

func UpdateConfig(c Config, field string) error {
	if field == "" {
		err := updateIfNotTheSame(YoutubeKeyUser, youtubeApiKeyEm.NotifySubscribers, cfg.YoutubeApiKey, c.YoutubeApiKey)
		if err != nil {
			return err
		}
	} else {
		switch field {
		case YoutubeKeyUser:
			err := updateIfNotTheSame(YoutubeKeyUser, youtubeApiKeyEm.NotifySubscribers, cfg.YoutubeApiKey, c.YoutubeApiKey)
			if err != nil {
				return err
			}
			//case obsSecretUser:
		}
	}

	cfg = c
	return nil
}

func migrateField(field string, key string) error {
	currentKey, err := getFromKeyring(field)
	if err != nil {
		return err
	}

	if currentKey != key {
		err := keyring.Set(custom_fyne.AppName, field, key)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateIfNotTheSame(field string, notifier func(string), old, new string) error {
	if old != new {
		err := keyring.Set(custom_fyne.AppName, field, new)
		if err != nil {
			return err
		}
		notifier(new)
	}
	return nil
}

func Get(field string) string {
	switch field {
	case YoutubeKeyUser:
		return cfg.YoutubeApiKey
	}
	return ""
}

func Subscribe(field string) *utils.EventSubscriber[string] {
	switch field {
	case YoutubeKeyUser:
		return youtubeApiKeyEm.SubscribeEvent()
	}
	return nil
}

func getFromKeyring(field string) (string, error) {
	content, err := keyring.Get(custom_fyne.AppName, field)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return content, err
}
