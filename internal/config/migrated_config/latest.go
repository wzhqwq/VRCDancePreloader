package migrated_config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Config Migration")

var MigrationNotes []string

func LoadLatestConfig() (config.Config, bool) {
	cfg := config.Default()

	if config.LoadConfigV3(&cfg) {
		return cfg, false
	}

	_, err := os.Stat("config.yaml")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.ErrorLn("Failed to open old config.yaml:", err)
			logger.WarnLn("Use default config instead.")
		}
		return cfg, false
	}

	err = MigrateFromV2(&cfg)
	if err != nil {
		logger.ErrorLn("Failed to migrate old config.yaml:", err)
		logger.WarnLn("Config might not be migrated completely.")
		MigrationNotes = append(MigrationNotes, i18n.T("message_migration_failure", goeasyi18n.Options{
			Data: map[string]interface{}{
				"Errors": err.Error(),
			},
		}))
		return cfg, false
	}

	err = config.SaveFileAtomic(cfg)
	if err != nil {
		logger.ErrorLn("Failed to save new config.yaml:", err)
	} else {
		logger.InfoLn("Migrated config.yaml successfully. The original config is still at current working directory.")
		MigrationNotes = append(MigrationNotes, i18n.T("message_config_v2_migration", goeasyi18n.Options{
			Data: map[string]interface{}{
				"Dir": filepath.Join(custom_fyne.AppConfigRoot, "app.yaml"),
			},
		}))
	}

	return cfg, true
}
