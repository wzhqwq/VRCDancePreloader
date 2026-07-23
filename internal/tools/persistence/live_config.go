package persistence

import (
	"database/sql"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence/db_vc"
)

var liveConfigTable = db_vc.DefTable("live_config").DefColumns(
	db_vc.NewText("web_version").SetPrimary(),
	db_vc.NewText("web_config"),
)

var setConfig = liveConfigTable.InsertOrReplace("web_version", "web_config").Build()
var getConfig = liveConfigTable.Select("web_config").Where("web_version = ?").Build()

func GetLiveConfig(webVersion string) string {
	row := liveConfigTable.QueryRow(getConfig, webVersion)

	var config string
	err := row.Scan(&config)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logger.ErrorLn("Failed to get live config:", err)
		}
		return "{}"
	}

	return config
}

func SetLiveConfig(webVersion string, config string) {
	_, err := liveConfigTable.Exec(setConfig, webVersion, config)
	if err != nil {
		logger.ErrorLn("Failed to set live config:", err)
	}
}
