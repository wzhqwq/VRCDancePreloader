package db_vc

import (
	"database/sql"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var compatibilityTable = DefTable("compatibility").DefColumns(
	NewText("key").SetPrimary(),
	NewText("value"),
)

var minimumDataVersion utils.ShortVersion
var maximumDataVersion utils.ShortVersion
var currentDataVersion utils.ShortVersion

func InitCompatibility(db *sql.DB) {
	_, compatibilityExists := tableNames[compatibilityTable.name]
	err := compatibilityTable.Init(db, !compatibilityExists)
	if err != nil {
		logger.FatalLn("Failed to initialize compatibility table", err)
	}

	minimumDataVersion = GetVersionInComp("minimumDataVersion", currentDataVersion)
	maximumDataVersion = GetVersionInComp("maximumDataVersion", currentDataVersion)
}

var queryValue = compatibilityTable.Select("value").Where("key = ?").Build()
var insertKV = compatibilityTable.Insert("key", "value").Build()
var updateValue = compatibilityTable.Update().Set("value = ?").Where("key = ?").Build()

func GetVersionInComp(key string, defaultVer utils.ShortVersion) utils.ShortVersion {
	row := compatibilityTable.QueryRow(queryValue, key)

	var value string
	err := row.Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = compatibilityTable.Exec(insertKV, key, defaultVer.String())
		if err != nil {
			logger.FatalLn("Failed to set local data version", key, err)
		}
		return defaultVer
	}

	if err != nil {
		logger.FatalLn("Failed to get local data version", key, err)
	}

	ver, ok := utils.ParseShortVersion(value)
	if !ok {
		SetVersionInComp(key, defaultVer)
		return defaultVer
	}

	return ver
}

func SetVersionInComp(key string, ver utils.ShortVersion) {
	_, err := compatibilityTable.Exec(updateValue, ver.String(), key)
	if err != nil {
		logger.FatalLn("Failed to set local data version", key, err)
	}
}
