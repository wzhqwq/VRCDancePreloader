package persistence

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"sync"
)

const allowListTableSQL = `
CREATE TABLE IF NOT EXISTS allow_list (
    		id TEXT PRIMARY KEY,
    		size INTEGER
);
`

type AllowList struct {
	sync.Mutex
	Entries map[string]int64

	em *utils.StringEventManager
}

var currentAllowList *AllowList

func (a *AllowList) addEntry(id string, size int64) {
	query := "INSERT INTO allow_list (id, size) VALUES (?, ?)"
	_, err := DB.Exec(query, id, size)
	if err != nil {
		log.Println("failed to save allow list entry:", err)
		return
	}

	a.Entries[id] = size
}

func (a *AllowList) getEntry(id string) int64 {
	row := DB.QueryRow("SELECT size FROM allow_list WHERE id = ?", id)

	var size int64
	err := row.Scan(&size)
	if err != nil {
		return 0
	}

	return size
}

func (a *AllowList) removeEntry(id string) {
	query := "DELETE FROM allow_list WHERE id = ?"
	_, err := DB.Exec(query, id)
	if err != nil {
		log.Println("failed to remove allow list entry:", err)
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
	rows, err := DB.Query("SELECT id, size FROM allow_list")
	if err != nil {
		log.Println("failed to load allow list:", err)
		return nil
	}
	defer rows.Close()

	var entries []types.CacheFileInfo
	for rows.Next() {
		var entry types.CacheFileInfo
		err := rows.Scan(&entry.ID, &entry.Size)
		if err != nil {
			log.Println("failed to load allow list entry:", err)
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

	rows, err := DB.Query("SELECT id, size FROM allow_list")
	if err != nil {
		log.Println("failed to load allow list:", err)
		return err
	}

	for rows.Next() {
		var id string
		var size int64
		err = rows.Scan(&id, &size)
		if err != nil {
			log.Println("failed to load allow list entry:", err)
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
		em:      utils.NewStringEventManager(),
	}

	err := currentAllowList.LoadEntries()
	if err != nil {
		log.Println("failed to load allow list:", err)
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
