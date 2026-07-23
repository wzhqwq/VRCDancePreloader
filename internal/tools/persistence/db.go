package persistence

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence/db_vc"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var MainDB *sql.DB

var logger = utils.NewLogger("DB")

var dataVersion = utils.ShortVersion{
	Major: 1,
	Minor: 1,
}

func InitMainDB() error {
	params := url.Values{}
	params.Add("_journal_mode", "WAL")
	params.Add("_synchronous", "NORMAL")
	params.Add("_temp_store", "MEMORY")

	dbFilePath := filepath.Join(custom_fyne.AppDataRoot, "db", "data.db")

	err := os.MkdirAll(filepath.Dir(dbFilePath), 0777)
	if err != nil {
		return err
	}

	MainDB, err = sql.Open("sqlite3", dbFilePath+"?"+params.Encode())
	if err != nil {
		return err
	}

	db_vc.Init(
		MainDB, dataVersion,
		localSongTable,
		danceRecordTable,
		allowListTable,
		cacheMetaTable,
		scheduleTable,
		liveConfigTable,
	)

	InitLocalSongs()
	InitAllowList()
	InitLocalRecords()
	return nil
}

func WalCheckpoint() {
	if MainDB == nil {
		return
	}

	_, err := MainDB.Exec("PRAGMA wal_checkpoint(PASSIVE);")
	if err != nil {
		logger.ErrorLn("Failed to set WAL checkpoint:", err)
	}
}

func CloseMainDB() error {
	return MainDB.Close()
}
