package persistence

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vs"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var DB *sql.DB

var logger = utils.NewLogger("DB")

var dataVersion = utils.ShortVersion{
	Major: 1,
	Minor: 0,
}

func InitDB(dbFilePath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return err
	}

	db_vs.Init(
		DB, dataVersion,
		localSongTable,
		danceRecordTable,
		allowListTable,
		worldDataTable,
	)

	_, err = DB.Exec(worldDataTableSQL)
	if err != nil {
		return err
	}

	InitLocalSongs()
	InitAllowList()
	InitLocalRecords()
	return nil
}

func CloseDB() {
	DB.Close()
}
