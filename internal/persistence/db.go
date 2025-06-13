package persistence

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dbFilePath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return err
	}

	_, err = DB.Exec(allowListTableSQL)
	if err != nil {
		return err
	}

	_, err = DB.Exec(danceRecordTableSQL)
	if err != nil {
		return err
	}

	_, err = DB.Exec(localSongTableSQL)
	if err != nil {
		return err
	}
	for _, query := range localSongTableIndicesSQLs {
		_, err = DB.Exec(query)
		if err != nil {
			return err
		}
	}

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
