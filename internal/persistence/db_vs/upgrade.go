package db_vs

import (
	"database/sql"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("DB VerCtrl")

func Init(db *sql.DB, dataVersion utils.ShortVersion, tables ...*Table) {
	getTableNames(db)
	currentDataVersion = dataVersion

	InitCompatibility(db)
	if minimumDataVersion.NewerThan(dataVersion) {
		logger.FatalLn("This software is incompatible with local database. Please upgrade the software")
	}

	upgradeNeeded := dataVersion.NewerThan(maximumDataVersion)
	if upgradeNeeded {
		logger.InfoLn("Upgrading database. Wait a minute...")
	}
	for _, t := range tables {
		err := t.Init(db, upgradeNeeded)
		if errors.Is(err, ErrUpgradeNeeded) {
			logger.WarnLn("The maximum compatible version of database is incorrect. Upgrading")
			upgradeNeeded = true
			err = t.Init(db, true)
		}
		if err != nil {
			logger.FatalLn("Failed to initialize or upgrade table", t.name, err)
		}
	}
	if upgradeNeeded {
		logger.InfoLn("Upgraded database", maximumDataVersion.String(), "->", dataVersion.String())
	}

	logger.InfoLn("Software data version:", dataVersion.String())
	logger.InfoLn("Database compatibility:", minimumDataVersion.String(), "~", maximumDataVersion.String())
}
