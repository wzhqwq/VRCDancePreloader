package persistence

import (
	"database/sql"
	"errors"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vc"
)

var scheduleTable = db_vc.DefTable("schedule").DefColumns(
	db_vc.NewTextId(),
	db_vc.NewInt("time"),
)

var addSchedule = scheduleTable.Insert("id", "time").Build()
var getSchedule = scheduleTable.Select("time").Where("id = ?").Build()
var setSchedule = scheduleTable.Update().Set("time = ?").Where("id = ?").Build()

func GetSchedule(name string) time.Time {
	row := scheduleTable.QueryRow(getSchedule, name)

	var t int64
	err := row.Scan(&t)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}
		}
		logger.ErrorLn("Failed to get schedule:", err)
	}

	return time.Unix(t, 0)
}

func SetSchedule(name string, t time.Time) {
	tx, err := scheduleTable.Tx()
	if err != nil {
		logger.ErrorLn("Failed to set schedule:", err)
		return
	}
	defer tx.Rollback()

	rows, err := tx.Query(getSchedule, name)
	if err != nil {
		logger.ErrorLn("Failed to check schedule:", err)
		return
	}
	if rows.Next() {
		_, err = tx.Exec(setSchedule, t.Unix(), name)
		if err != nil {
			logger.ErrorLn("Failed to set schedule:", err)
		}
	} else {
		_, err = tx.Exec(addSchedule, name, t.Unix())
		if err != nil {
			logger.ErrorLn("Failed to set schedule:", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		logger.ErrorLn("Failed to set schedule:", err)
	}
}

func IsScheduleDueReached(name string, interval time.Duration) bool {
	t := GetSchedule(name)
	return t.IsZero() || time.Now().After(t.Add(interval))
}

func UpdateSchedule(name string) {
	SetSchedule(name, time.Now())
}
