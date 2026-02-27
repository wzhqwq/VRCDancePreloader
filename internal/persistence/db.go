package persistence

import (
	"database/sql"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vc"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var DB *sql.DB

var logger = utils.NewLogger("DB")

var dataVersion = utils.ShortVersion{
	Major: 1,
	Minor: 0,
}

func InitDB(dbFilePath string) error {
	params := url.Values{}
	params.Add("_journal_mode", "WAL")
	params.Add("_synchronous", "NORMAL")
	params.Add("_temp_store", "MEMORY")

	var err error
	DB, err = sql.Open("sqlite3", dbFilePath+"?"+params.Encode())
	if err != nil {
		return err
	}

	db_vc.Init(
		DB, dataVersion,
		localSongTable,
		danceRecordTable,
		allowListTable,
		worldDataTable,
		cacheMetaTable,
		scheduleTable,
	)

	InitLocalSongs()
	InitAllowList()
	InitLocalRecords()
	return nil
}

func WalCheckpoint() {
	if DB == nil {
		return
	}

	_, err := DB.Exec("PRAGMA wal_checkpoint(PASSIVE);")
	if err != nil {
		logger.ErrorLn("Failed to set WAL checkpoint:", err)
	}
}

func CloseDB() {
	DB.Close()
}
