package cache

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

const rebuildThreshold = 10000

// <type>$<name>.<ext>
var modernCacheNameRegex = regexp.MustCompile(`^(.+)\$([^.]+)\.(.+)$`)

type fileInfo struct {
	id      string
	size    int64
	partial bool
	created time.Time
}

func SyncWithFS() error {
	allInfo, err := cache_fs.Scan()
	if err != nil {
		return err
	}

	listByType := make(map[string][]fileInfo)
	for _, info := range allInfo {
		matches := modernCacheNameRegex.FindStringSubmatch(info.Name)
		if matches != nil {
			listByType[matches[1]] = append(listByType[matches[1]], fileInfo{
				id:      matches[2],
				size:    info.Size,
				partial: strings.HasSuffix(matches[3], ".dl"),
				created: info.Created,
			})
		}
	}

	tx, err := persistence.BeginCacheSyncTx()
	if err != nil {
		return err
	}
	// full ACID guarantee
	defer tx.Abort()

	var itemsToInsert []*persistence.CacheMeta
	var itemsToDelete []string

	diffCount := 0
	rebuildNeeded := false

	for fileType, list := range listByType {
		// sort by name
		idsInDb, err := persistence.ListIDsByType(fileType)
		if err != nil {
			return err
		}

		slices.SortFunc(list, func(a, b fileInfo) int {
			return strings.Compare(a.id, b.id)
		})

		localIdx := 0
		localLen := len(list)

		idsInDb = append(idsInDb, "")

		for _, id := range idsInDb {
			for localIdx < localLen && (id == "" || list[localIdx].id < id) {
				item := list[localIdx]
				meta := persistence.NewCacheMeta(item.id, fileType, item.size, time.Time{}, item.created)
				meta.Partial = item.partial
				itemsToInsert = append(itemsToInsert, meta)
				localIdx++
				diffCount++
			}

			if id == "" {
				break
			}

			if localIdx < localLen && list[localIdx].id == id {
				localIdx++
			} else {
				itemsToDelete = append(itemsToDelete, fileType+"$"+id)
				diffCount++
			}

			if diffCount > rebuildThreshold {
				rebuildNeeded = true
				break
			}
		}
	}

	if rebuildNeeded {
		managerLogger.WarnLn("There are too many differences between the database and the file system, rebuilding the database")
		var records []*persistence.CacheMeta
		for fileType, list := range listByType {
			for _, item := range list {
				meta := persistence.NewCacheMeta(item.id, fileType, item.size, time.Time{}, item.created)
				meta.Partial = item.partial
				records = append(records, meta)
			}
		}

		return tx.Rebuild(records)
	}

	return tx.Diff(itemsToInsert, itemsToDelete)
}
