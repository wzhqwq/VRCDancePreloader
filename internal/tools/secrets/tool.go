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
	keyChanged := false

	if field == "" || field == YoutubeKeyUser {
		changed, err := updateIfNotTheSame(YoutubeKeyUser, cfg.YoutubeApiKey, c.YoutubeApiKey)
		if err != nil {
			return err
		}
		keyChanged = changed
		// case obsSecretUser:
	}

	// Stored before the notification: the readers recompute from the current
	// state when they handle the event (third_parties' gates read Get() instead
	// of using the payload), so the key they then find has to be the announced
	// one.
	cfg = c

	if keyChanged {
		youtubeApiKeyEm.NotifySubscribers(c.YoutubeApiKey)
	}

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

// updateIfNotTheSame writes the new secret into the keyring and reports whether
// it changed at all. The caller notifies only after it stored the new config, so
// that a reader woken by the notification cannot see the previous value.
func updateIfNotTheSame(field string, old, new string) (bool, error) {
	if old == new {
		return false, nil
	}

	if err := keyring.Set(custom_fyne.AppName, field, new); err != nil {
		return false, err
	}

	return true, nil
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
