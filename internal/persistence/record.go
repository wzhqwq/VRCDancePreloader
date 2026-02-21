package persistence

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vc"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var localRecords *LocalRecords
var currentRoomName string

var danceRecordTable = db_vc.DefTable("dance_record").DefColumns(
	db_vc.NewIncreasingId(),
	db_vc.NewInt("start_time"),
	db_vc.NewText("comment"),
	db_vc.NewText("orders"),
)

var danceRecordColumns = []string{"id", "start_time", "comment", "orders"}

var insertRecord = danceRecordTable.Insert(danceRecordColumns[1:]...).Build()

var listSimplifiedRecords = danceRecordTable.Select("id", "start_time").Sort("start_time", false).Build()
var getRecord = danceRecordTable.Select(danceRecordColumns...).Where("id = ?").Build()
var getLatestRecord = danceRecordTable.Select(danceRecordColumns...).Sort("start_time", false).Limit(1).Build()

var setOrders = danceRecordTable.Update().Set("orders = ?").Where("id = ?").Build()
var setComment = danceRecordTable.Update().Set("comment = ?").Where("id = ?").Build()

var deleteRecord = danceRecordTable.Delete().Where("id = ?").Build()

type DanceRecord struct {
	ID        int
	StartTime time.Time
	Comment   string
	Orders    []Order

	em *utils.EventManager[string]

	ordersMutex sync.RWMutex
}

type SimplifiedDanceRecord struct {
	ID        int
	StartTime time.Time
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

		ordersMutex: sync.RWMutex{},
	}
}

func NewDanceRecordFromScan(row *sql.Row) (*DanceRecord, error) {
	record := NewEmptyDanceRecord()
	var orders string
	var startTimeInt int64
	if err := row.Scan(&record.ID, &startTimeInt, &record.Comment, &orders); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("record not found")
		}
		return nil, err
	}
	record.StartTime = time.Unix(startTimeInt, 0)
	err := json.Unmarshal([]byte(orders), &record.Orders)
	if err != nil {
		logger.ErrorLn("Error unmarshalling dance record: ", err)
	}
	return record, nil
}

func (r *DanceRecord) AddOrder(order Order) {
	r.doAdd(order)
	r.em.NotifySubscribers("+" + order.ID)
	if r.ID != -1 {
		r.updateOrders()
	} else {
		err := localRecords.addRecord(r)
		if err != nil {
			logger.ErrorLn("error adding record:", err)
		}
	}
}

func (r *DanceRecord) doAdd(order Order) {
	r.ordersMutex.Lock()
	defer r.ordersMutex.Unlock()

	if len(r.Orders) > 0 {
		lastOrder := r.Orders[len(r.Orders)-1]
		if lastOrder.ID == order.ID && math.Abs(float64(lastOrder.Time.Unix()-order.Time.Unix())) < 60 {
			return
		}
	}

	r.Orders = append(r.Orders, order)
}

func (r *DanceRecord) RemoveOrder(orderTime time.Time) {
	if id := r.doRemove(orderTime); id != "" {
		r.em.NotifySubscribers("-" + id)
		if r.ID != -1 {
			r.updateOrders()
		}
	}
}

func (r *DanceRecord) doRemove(orderTime time.Time) string {
	r.ordersMutex.Lock()
	defer r.ordersMutex.Unlock()

	removeIndex := -1
	for i, order := range r.Orders {
		if order.Time.Unix() == orderTime.Unix() {
			removeIndex = i
			break
		}
	}

	if removeIndex == -1 {
		return ""
	}
	order := r.Orders[removeIndex]
	r.Orders = append(r.Orders[:removeIndex], r.Orders[removeIndex+1:]...)

	return order.ID
}

func (r *DanceRecord) GetOrdersSnapshot() []Order {
	r.ordersMutex.RLock()
	orders := make([]Order, len(r.Orders))
	copy(orders, r.Orders)
	r.ordersMutex.RUnlock()

	return orders
}

func (r *DanceRecord) SetComment(comment string) {
	r.Comment = comment
	if r.ID != -1 {
		r.updateComment()
	}
}

func (r *DanceRecord) updateOrders() {
	data, err := json.Marshal(r.GetOrdersSnapshot())
	if err != nil {
		return
	}

	_, err = danceRecordTable.Exec(setOrders, string(data), r.ID)
}

func (r *DanceRecord) updateComment() {
	_, err := danceRecordTable.Exec(setComment, r.Comment, r.ID)
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

func (o Order) Key() string {
	return o.ID + "_" + o.Time.Format("15:04:05")
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
		existingRecord.Comment = record.Comment

		existingRecord.ordersMutex.Lock()
		existingRecord.Orders = record.Orders
		existingRecord.ordersMutex.Unlock()

		return existingRecord
	}

	l.Records[record.ID] = record
	return record
}

func (l *LocalRecords) GetRecords() ([]*SimplifiedDanceRecord, error) {
	rows, err := danceRecordTable.Query(listSimplifiedRecords)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*SimplifiedDanceRecord
	for rows.Next() {
		record := &SimplifiedDanceRecord{}
		var startTimeInt int64
		if err := rows.Scan(&record.ID, &startTimeInt); err != nil {
			return nil, err
		}
		record.StartTime = time.Unix(startTimeInt, 0)

		records = append(records, record)
	}

	return records, nil
}

func (l *LocalRecords) GetRecord(id int) (*DanceRecord, error) {
	rows := danceRecordTable.QueryRow(getRecord, id)

	record, err := NewDanceRecordFromScan(rows)
	if err != nil {
		return nil, err
	}

	return l.ReplaceIfExists(record), nil
}

func (l *LocalRecords) DeleteRecord(id int) error {
	_, err := danceRecordTable.Exec(deleteRecord, id)
	if err != nil {
		return err
	}
	l.em.NotifySubscribers("-" + strconv.Itoa(id))
	return nil
}

func (l *LocalRecords) addRecord(r *DanceRecord) error {
	data, err := json.Marshal(r.GetOrdersSnapshot())
	if err != nil {
		return err
	}

	result, err := danceRecordTable.Exec(insertRecord, r.StartTime.Unix(), r.Comment, string(data))
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
	rows := danceRecordTable.QueryRow(getLatestRecord)

	record, err := NewDanceRecordFromScan(rows)
	if err != nil {
		return nil, err
	}

	return l.ReplaceIfExists(record), nil
}

func (l *LocalRecords) GetNearestRecord() *DanceRecord {
	latestRecord, _ := l.getLatestRecord()
	if latestRecord == nil {
		return nil
	}

	orders := latestRecord.GetOrdersSnapshot()
	lastOrder := orders[len(orders)-1]
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
