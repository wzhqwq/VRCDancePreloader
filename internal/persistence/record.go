package persistence

import (
	"encoding/json"
	"time"
)

var currentRecord *DanceRecord

const danceRecordTableSQL = `
CREATE TABLE IF NOT EXISTS dance_record (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time TEXT,
		comment TEXT,
		orders TEXT
);
`

type DanceRecord struct {
	ID        int
	StartTime string
	Comment   string
	Orders    []Order

	InDB bool
}

func (r *DanceRecord) AddOrder(order Order) {
	r.Orders = append(r.Orders, order)
	if r.InDB {
		r.update()
	}
}

func (r *DanceRecord) save() {
	data, err := json.Marshal(r.Orders)
	if err != nil {
		return
	}

	_, err = DB.Exec("INSERT INTO dance_record (start_time, orders) VALUES (?, ?)", r.StartTime, string(data))
}

func (r *DanceRecord) update() {
	data, err := json.Marshal(r.Orders)
	if err != nil {
		return
	}

	_, err = DB.Exec("UPDATE dance_record SET orders = ? WHERE id = ?", string(data), r.ID)
}

type Order struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Username string `json:"username"`
	Time     string `json:"time"`
}

func AddToHistory(id, title, username, time string) {
	currentRecord.AddOrder(Order{
		ID:       id,
		Title:    title,
		Username: username,
		Time:     time,
	})
}

func PrepareHistory() {
	currentRecord = &DanceRecord{
		StartTime: time.Now().Format("2006-01-02 15:04:05"),
		Comment:   "",
		Orders:    make([]Order, 0),
	}
}

func SaveHistory() {
	currentRecord.save()
}
