package secrets

import (
	"errors"
	"sync/atomic"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/zalando/go-keyring"
)

const YoutubeKeyUser = "youtube-api-key"
const ObsSecretUser = "obs-websocket"

// cfg is read from arbitrary goroutines (the providers' loops, the request path)
// and written by the configuration thread, hence the atomic pointer.
var cfg atomic.Pointer[Config]

// youtubeApiKey is the level the YouTube provider watches: wait on a watcher's
// Wakes, then read Get for the key. It replaces the former "notify with the new
// value" event, so a reader cannot see a value older than the wake-up that woke
// it. Every consumer gets its own watcher from Source.
var youtubeApiKey = utils.NewLevel("")

// neverChanges is what Source hands out for a field nobody publishes: a watcher
// that is never woken instead of a nil the caller would wait on forever.
var neverChanges = utils.NewLevel("")

// unknownFieldWatcher is shared on purpose: it never wakes, so sharing it is
// harmless and avoids leaking a watcher per call.
var unknownFieldWatcher = neverChanges.Subscribe()

type Tool struct {
	service.ConfigurableTool
}

func New(c Config) *Tool {
	cfg.Store(&c)

	youtubeApiKey.Store(c.YoutubeApiKey)

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

	current := cfg.Load()
	previousKey := ""
	if current != nil {
		previousKey = current.YoutubeApiKey
	}

	if field == "" || field == YoutubeKeyUser {
		changed, err := updateIfNotTheSame(YoutubeKeyUser, previousKey, c.YoutubeApiKey)
		if err != nil {
			return err
		}
		keyChanged = changed
		// case obsSecretUser:
	}

	// Stored before the wake-up (utils.Level does that in one call): the readers
	// re-read Get when they handle the wake-up, so the key they then find has to
	// be the announced one.
	cfg.Store(&c)
	if keyChanged {
		youtubeApiKey.Store(c.YoutubeApiKey)
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
// it changed at all. The caller publishes the new config (and wakes the readers)
// only after it stored the new value.
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
	current := cfg.Load()
	if current == nil {
		return ""
	}

	switch field {
	case YoutubeKeyUser:
		return current.YoutubeApiKey
	}
	return ""
}

// Source is the watcher a consumer uses for a field: wait on Wakes, then read
// Get for the value. It returns the concrete watcher on purpose — the consumer
// has to Close it when it stops, so that the level does not keep waking a
// consumer that is gone. An unknown field yields a shared watcher that never
// wakes rather than a nil the caller would wait on forever.
func Source(field string) *utils.LevelWatcher[string] {
	switch field {
	case YoutubeKeyUser:
		return youtubeApiKey.Subscribe()
	}
	return unknownFieldWatcher
}

func getFromKeyring(field string) (string, error) {
	content, err := keyring.Get(custom_fyne.AppName, field)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return content, err
}
