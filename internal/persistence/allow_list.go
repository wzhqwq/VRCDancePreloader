package persistence

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vc"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

var allowListTable = db_vc.DefTable("allow_list").DefColumns(
	db_vc.NewTextId(),
	db_vc.NewInt("size"),
)

var listAllowList = allowListTable.Select("id", "size").Build()

type AllowList struct {
	sync.Mutex
	Entries map[string]int64
}

var currentAllowList *AllowList

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

	return nil
}

func InitAllowList() {
	currentAllowList = &AllowList{
		Entries: make(map[string]int64),
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

func GetAllowListEntries() []types.CacheFileInfo {
	return currentAllowList.GetAllowList()
}
