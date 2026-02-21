package persistence

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vc"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var allowListTable = db_vc.DefTable("allow_list").DefColumns(
	db_vc.NewTextId(),
	db_vc.NewInt("size"),
)

var listAllowList = allowListTable.Select("id", "size").Build()

var insertAllowEntry = allowListTable.Insert("id", "size").Build()

var deleteAllowEntry = allowListTable.Delete().Where("id = ?").Build()

type AllowList struct {
	sync.Mutex
	Entries map[string]int64

	em *utils.EventManager[string]
}

var currentAllowList *AllowList

func (a *AllowList) addEntry(id string, size int64) {
	_, err := allowListTable.Exec(insertAllowEntry, id, size)
	if err != nil {
		logger.ErrorLn("Failed to save allow list entry:", err)
		return
	}

	a.Entries[id] = size
}

func (a *AllowList) removeEntry(id string) {
	_, err := allowListTable.Exec(deleteAllowEntry, id)
	if err != nil {
		logger.ErrorLn("Failed to remove allow list entry:", err)
		return
	}

	delete(a.Entries, id)
}

func (a *AllowList) AddToAllowList(id string, size int64) {
	a.Lock()
	defer a.Unlock()

	if _, ok := a.Entries[id]; ok {
		return
	}

	a.addEntry(id, size)
	a.em.NotifySubscribers("+" + id)
}

func (a *AllowList) RemoveFromAllowList(id string) {
	a.Lock()
	defer a.Unlock()

	if _, ok := a.Entries[id]; !ok {
		return
	}

	a.removeEntry(id)
	a.em.NotifySubscribers("-" + id)
}

func (a *AllowList) GetAllowList() []types.CacheFileInfo {
	rows, err := allowListTable.Query(listAllowList)
	if err != nil {
		logger.ErrorLn("Failed to load allow list:", err)
		return nil
	}
	defer rows.Close()

	var entries []types.CacheFileInfo
	for rows.Next() {
		var entry types.CacheFileInfo
		err := rows.Scan(&entry.ID, &entry.Size)
		if err != nil {
			logger.ErrorLn("Failed to load allow list entry:", err)
			continue
		}

		entries = append(entries, entry)
	}

	return entries
}

func (a *AllowList) IsInAllowList(id string) bool {
	a.Lock()
	defer a.Unlock()

	_, ok := a.Entries[id]
	return ok
}

func (a *AllowList) LoadEntries() error {
	a.Lock()
	defer a.Unlock()

	rows, err := allowListTable.Query(listAllowList)
	if err != nil {
		logger.ErrorLn("Failed to load allow list:", err)
		return err
	}

	for rows.Next() {
		var id string
		var size int64
		err = rows.Scan(&id, &size)
		if err != nil {
			logger.ErrorLn("Failed to load allow list entry:", err)
			continue
		}

		a.Entries[id] = size
	}

	a.em.NotifySubscribers("load")

	return nil
}

func InitAllowList() {
	currentAllowList = &AllowList{
		Entries: make(map[string]int64),
		em:      utils.NewEventManager[string](),
	}

	err := currentAllowList.LoadEntries()
	if err != nil {
		logger.ErrorLn("Failed to load allow list:", err)
	}
}

func GetAllowList() *AllowList {
	return currentAllowList
}

func IsInAllowList(id string) bool {
	return currentAllowList.IsInAllowList(id)
}

func AddToAllowList(id string, size int64) {
	currentAllowList.AddToAllowList(id, size)
}

func RemoveFromAllowList(id string) {
	currentAllowList.RemoveFromAllowList(id)
}

func GetAllowListEntries() []types.CacheFileInfo {
	return currentAllowList.GetAllowList()
}
