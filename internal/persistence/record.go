package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"math"
	"strconv"
	"sync"
	"time"
)

var localRecords *LocalRecords
var currentRoomName string

const danceRecordTableSQL = `
CREATE TABLE IF NOT EXISTS dance_record (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time INTEGER,
		comment TEXT,
		orders TEXT
);
`

type DanceRecord struct {
	sync.Mutex

	ID        int
	StartTime time.Time
	Comment   string
	Orders    []Order

	em *utils.EventManager[string]
}

func NewEmptyDanceRecord() *DanceRecord {
	return &DanceRecord{
		em: utils.NewEventManager[string](),
	}
}

func NewDanceRecord() *DanceRecord {
	return &DanceRecord{
		ID:        -1,
		StartTime: time.Now(),
		Comment:   "",
		Orders:    make([]Order, 0),

		em: utils.NewEventManager[string](),
	}
}

func NewDanceRecordFromScan(rows *sql.Rows) (*DanceRecord, error) {
	record := NewEmptyDanceRecord()
	var orders string
	var startTimeInt int64
	if err := rows.Scan(&record.ID, &startTimeInt, &record.Comment, &orders); err != nil {
		return nil, err
	}
	record.StartTime = time.Unix(startTimeInt, 0)
	json.Unmarshal([]byte(orders), &record.Orders)
	return record, nil
}

func (r *DanceRecord) AddOrder(order Order) {
	r.Lock()
	defer r.Unlock()

	if len(r.Orders) > 0 {
		lastOrder := r.Orders[len(r.Orders)-1]
		if lastOrder.ID == order.ID && math.Abs(float64(lastOrder.Time.Unix()-order.Time.Unix())) < 60 {
			return
		}
	}

	r.Orders = append(r.Orders, order)
	r.em.NotifySubscribers("+" + order.ID)
	if r.ID != -1 {
		r.updateOrders()
	} else {
		localRecords.addRecord(r)
	}
}

func (r *DanceRecord) RemoveOrder(orderTime time.Time) {
	r.Lock()
	defer r.Unlock()

	removeIndex := -1
	for i, order := range r.Orders {
		if order.Time.Unix() == orderTime.Unix() {
			removeIndex = i
			break
		}
	}

	if removeIndex == -1 {
		return
	}
	order := r.Orders[removeIndex]
	r.Orders = append(r.Orders[:removeIndex], r.Orders[removeIndex+1:]...)

	r.em.NotifySubscribers("-" + order.ID)
	if r.ID != -1 {
		r.updateOrders()
	}
}

func (r *DanceRecord) SetComment(comment string) {
	r.Comment = comment
	if r.ID != -1 {
		r.updateComment()
	}
}

func (r *DanceRecord) updateOrders() {
	data, err := json.Marshal(r.Orders)
	if err != nil {
		return
	}

	_, err = DB.Exec("UPDATE dance_record SET orders = ? WHERE id = ?", string(data), r.ID)
}

func (r *DanceRecord) updateComment() {
	_, err := DB.Exec("UPDATE dance_record SET comment = ? WHERE id = ?", r.Comment, r.ID)
	if err != nil {
		return
	}
}

func (r *DanceRecord) SubscribeEvent() *utils.EventSubscriber[string] {
	return r.em.SubscribeEvent()
}

type Order struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Username  string    `json:"username"`
	Time      time.Time `json:"time"`
	DanceRoom string    `json:"dance_room"`
}

func AddToHistory(id, title, username string, time time.Time) {
	if localRecords.CurrentRecord == nil {
		return
	}
	localRecords.CurrentRecord.AddOrder(Order{
		ID:        id,
		Title:     title,
		Username:  username,
		Time:      time,
		DanceRoom: currentRoomName,
	})
}

type LocalRecords struct {
	sync.Mutex

	CurrentRecord *DanceRecord

	Records map[int]*DanceRecord

	em *utils.EventManager[string]
}

func (l *LocalRecords) ReplaceIfExists(record *DanceRecord) *DanceRecord {
	l.Lock()
	defer l.Unlock()

	if existingRecord, ok := l.Records[record.ID]; ok {
		existingRecord.Lock()
		defer existingRecord.Unlock()

		existingRecord.Comment = record.Comment
		existingRecord.Orders = record.Orders
		return existingRecord
	} else {
		l.Records[record.ID] = record
		return record
	}
}

func (l *LocalRecords) GetRecords() ([]*DanceRecord, error) {
	rows, err := DB.Query("SELECT id, start_time, comment, orders FROM dance_record ORDER BY start_time DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*DanceRecord
	for rows.Next() {
		record, err := NewDanceRecordFromScan(rows)
		if err != nil {
			return nil, err
		}

		records = append(records, l.ReplaceIfExists(record))
	}

	return records, nil
}

func (l *LocalRecords) GetRecord(id int) (*DanceRecord, error) {
	rows, err := DB.Query("SELECT id, start_time, comment, orders FROM dance_record WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		record, err := NewDanceRecordFromScan(rows)
		if err != nil {
			return nil, err
		}
		return l.ReplaceIfExists(record), nil
	}

	return nil, fmt.Errorf("record not found")
}

func (l *LocalRecords) DeleteRecord(id int) error {
	_, err := DB.Exec("DELETE FROM dance_record WHERE id = ?", id)
	if err != nil {
		return err
	}
	l.em.NotifySubscribers("-" + strconv.Itoa(id))
	return nil
}

func (l *LocalRecords) addRecord(r *DanceRecord) error {
	data, err := json.Marshal(r.Orders)
	if err != nil {
		return err
	}

	sql := "INSERT INTO dance_record (start_time, comment, orders) VALUES (?, ?, ?)"
	result, err := DB.Exec(sql, r.StartTime.Unix(), r.Comment, string(data))
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	r.ID = int(id)
	l.em.NotifySubscribers("+" + strconv.Itoa(r.ID))

	l.Records[r.ID] = r

	return nil
}

func (l *LocalRecords) SubscribeEvent() *utils.EventSubscriber[string] {
	return l.em.SubscribeEvent()
}

func (l *LocalRecords) getLatestRecord() (*DanceRecord, error) {
	// get the latest record, which has the highest start_time
	rows, err := DB.Query("SELECT id, start_time, comment, orders FROM dance_record ORDER BY start_time DESC LIMIT 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		record, err := NewDanceRecordFromScan(rows)
		if err != nil {
			return nil, err
		}
		return l.ReplaceIfExists(record), nil
	}

	return nil, nil
}

func (l *LocalRecords) GetNearestRecord() *DanceRecord {
	latestRecord, _ := l.getLatestRecord()
	if latestRecord == nil {
		return nil
	}

	lastOrder := latestRecord.Orders[len(latestRecord.Orders)-1]
	if time.Now().Unix()-lastOrder.Time.Unix() > 30*60 {
		return nil
	}

	return latestRecord
}

func (l *LocalRecords) Start(useLatest bool) {
	if useLatest {
		latestRecord, _ := l.getLatestRecord()
		if latestRecord != nil {
			l.CurrentRecord = latestRecord
			return
		}
	}
	l.CurrentRecord = NewDanceRecord()
}

func InitLocalRecords() {
	localRecords = &LocalRecords{
		Records: make(map[int]*DanceRecord),

		em: utils.NewEventManager[string](),
	}
}

func PrepareHistory(useLatest bool) {
	localRecords.Start(useLatest)
}

func GetCurrentRecord() *DanceRecord {
	return localRecords.CurrentRecord
}

func GetLocalRecords() *LocalRecords {
	return localRecords
}

func SetCurrentRoomName(roomName string) {
	currentRoomName = roomName
}
