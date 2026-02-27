package persistence

import (
	"database/sql"
	"errors"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/persistence/db_vc"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var cacheMetaTable = db_vc.DefTable("cache_meta").DefColumns(
	db_vc.NewText("name").SetPrimary(),
	db_vc.NewText("entity_id").SetIndexed(),
	db_vc.NewText("type").SetIndexed(),
	db_vc.NewInt("size").SetIndexed(),
	db_vc.NewBool("partial"),
	db_vc.NewBool("preserved").SetIndexed(),
	db_vc.NewInt("remote_last_modified"),
	db_vc.NewInt("created_time").SetIndexed(),
	db_vc.NewInt("last_accessed").SetIndexed(),
)

var cacheMetaColumns = []string{"name", "entity_id", "type", "size", "partial", "preserved", "remote_last_modified", "created_time", "last_accessed"}

var getMeta = cacheMetaTable.Select(cacheMetaColumns...).Where("name = ?").Build()
var listAllByType = cacheMetaTable.Select(cacheMetaColumns...).Where("type = ?").Paginate()
var listPreservedByType = cacheMetaTable.Select(cacheMetaColumns...).Where("type = ? AND preserved = true").Paginate()
var listSortedIDsByType = cacheMetaTable.Select("entity_id").Where("type = ?").Sort("entity_id", true).Build()
var sumOfSizeByType = cacheMetaTable.Select("SUM(*)").Where("type = ?").Build()
var sumOfSizeGroupedByType = cacheMetaTable.Select("type", "SUM(size)").Group("type").Build()
var listCleanupCandidates = cacheMetaTable.Select("entity_id", "size").Where("type = ? AND preserved = false").Sort("last_accessed", true).Build()

var addMeta = cacheMetaTable.Insert(cacheMetaColumns...).Build()

var setMetaLA = cacheMetaTable.Update().Set("last_accessed = ?").Where("name = ?").Build()
var setMetaPartial = cacheMetaTable.Update().Set("partial = ?").Where("name = ?").Build()
var setMetaPreserved = cacheMetaTable.Update().Set("preserved = ?").Where("name = ?").Build()
var setMetaInfo = cacheMetaTable.Update().Set(
	"size = ?", "remote_last_modified = ?", "created_time = ?",
).Where("name = ?").Build()

var deleteMeta = cacheMetaTable.Delete().Where("name = ?").Build()

type MetaChange struct {
	T    string
	ID   string
	Type string
}

var metaTableEm = utils.NewEventManager[MetaChange]()
var preservedListEm = utils.NewEventManager[MetaChange]()

type CacheMeta struct {
	// use name as primary key, we use video$<id>
	// TODO v3: <type>$<id>
	Name string

	// one entity might be related to multiple files (video, thumbnail, etc.)
	EntityID string
	// TODO v3: video, thumbnail, manifest, etc.
	Type string

	Size    int64
	Partial bool

	Preserved bool

	RemoteLastModified time.Time
	CreatedTime        time.Time
	LastAccessed       time.Time
}

func (c *CacheMeta) Access() {
	c.LastAccessed = time.Now()

	_, err := cacheMetaTable.Exec(setMetaLA, c.LastAccessed.Unix(), c.Name)
	if err != nil {
		logger.ErrorLn("Failed to update access time of", c.Name)
	}

	metaTableEm.NotifySubscribers(MetaChange{T: "*", ID: c.EntityID, Type: c.Type})
	if c.Preserved {
		preservedListEm.NotifySubscribers(MetaChange{T: "*", ID: c.EntityID, Type: c.Type})
	}
}

func (c *CacheMeta) SetPartial(partial bool) {
	c.Partial = partial

	_, err := cacheMetaTable.Exec(setMetaPartial, partial, c.Name)
	if err != nil {
		logger.ErrorLn("Failed to update the integrity of", c.Name)
	}

	metaTableEm.NotifySubscribers(MetaChange{T: "*", ID: c.EntityID, Type: c.Type})
	if c.Preserved {
		preservedListEm.NotifySubscribers(MetaChange{T: "*", ID: c.EntityID, Type: c.Type})
	}
}

func (c *CacheMeta) SetPreserved(preserved bool) {
	c.Preserved = preserved

	_, err := cacheMetaTable.Exec(setMetaPreserved, preserved, c.Name)
	if err != nil {
		logger.ErrorLn("Failed to update preserved state of", c.Name)
	}

	metaTableEm.NotifySubscribers(MetaChange{T: "*", ID: c.EntityID, Type: c.Type})
	if preserved {
		preservedListEm.NotifySubscribers(MetaChange{T: "+", ID: c.EntityID, Type: c.Type})
	} else {
		preservedListEm.NotifySubscribers(MetaChange{T: "-", ID: c.EntityID, Type: c.Type})
	}
}

func (c *CacheMeta) UpdateInfo(size int64, remoteLM, createdTime time.Time) error {
	c.Size = size
	c.RemoteLastModified = remoteLM
	c.CreatedTime = createdTime

	_, err := cacheMetaTable.Exec(setMetaInfo, size, remoteLM.Unix(), createdTime.Unix())
	if err != nil {
		return err
	}

	metaTableEm.NotifySubscribers(MetaChange{T: "*", ID: c.EntityID, Type: c.Type})
	if c.Preserved {
		preservedListEm.NotifySubscribers(MetaChange{T: "*", ID: c.EntityID, Type: c.Type})
	}
	return nil
}

func (c *CacheMeta) Delete(tx ...*sql.Tx) {
	var err error
	if len(tx) > 0 {
		_, err = tx[0].Exec(deleteMeta, c.Name)
	} else {
		_, err = cacheMetaTable.Exec(deleteMeta, c.Name)
	}
	if err != nil {
		logger.ErrorLn("Failed to remove cache meta", c.Name)
	}

	metaTableEm.NotifySubscribers(MetaChange{T: "-", ID: c.EntityID, Type: c.Type})
	if c.Preserved {
		preservedListEm.NotifySubscribers(MetaChange{T: "-", ID: c.EntityID, Type: c.Type})
	}
}

func NewCacheMeta(entityID, fileType string, size int64, remoteLM, createdTime time.Time) *CacheMeta {
	return &CacheMeta{
		Name:               fileType + "$" + entityID,
		EntityID:           entityID,
		Type:               fileType,
		Size:               size,
		RemoteLastModified: remoteLM,
		CreatedTime:        createdTime,
		LastAccessed:       createdTime,
	}
}

func AddCacheMeta(c *CacheMeta, tx ...*sql.Tx) error {
	var args = []any{
		c.Name, c.EntityID, c.Type, c.Size, c.Partial, c.Preserved,
		c.RemoteLastModified.Unix(), c.CreatedTime.Unix(), c.LastAccessed.Unix(),
	}

	var err error
	if len(tx) > 0 {
		_, err = tx[0].Exec(addMeta, args)
	} else {
		_, err = cacheMetaTable.Exec(addMeta, args)
	}
	if err != nil {
		return err
	}

	metaTableEm.NotifySubscribers(MetaChange{T: "+", ID: c.EntityID, Type: c.Type})
	if c.Preserved {
		preservedListEm.NotifySubscribers(MetaChange{T: "+", ID: c.EntityID, Type: c.Type})
	}
	return nil
}

func GetCacheMeta(entityID, fileType string, tx ...*sql.Tx) (*CacheMeta, bool) {
	var row *sql.Row
	if len(tx) > 0 {
		row = tx[0].QueryRow(getMeta, fileType+"$"+entityID)
	} else {
		row = cacheMetaTable.QueryRow(getMeta, fileType+"$"+entityID)
	}

	c := &CacheMeta{}
	var remoteLM, createdTime, lastAccessed int64
	if err := row.Scan(
		&c.Name, &c.EntityID, &c.Type, &c.Size, &c.Partial, &c.Preserved,
		&remoteLM, &createdTime, &lastAccessed,
	); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logger.ErrorLn("Failed to get cache meta of", fileType, entityID)
		}
		return nil, false
	}
	c.RemoteLastModified = time.Unix(remoteLM, 0)
	c.CreatedTime = time.Unix(createdTime, 0)
	c.LastAccessed = time.Unix(lastAccessed, 0)

	return c, true
}

func ListCacheMeta(fileType, sortColumn string, page, pageSize int, preserved bool) ([]*CacheMeta, error) {
	q := listAllByType
	if preserved {
		q = listPreservedByType
	}
	switch sortColumn {
	case "id":
		q.Sort("entity_id", true)
	case "size":
		q.Sort("size", false)
	case "created":
		q.Sort("created_time", true)
	case "accessed":
		q.Sort("last_accessed", true)
	}
	rows, err := cacheMetaTable.Query(q.Build(), fileType, pageSize, page*pageSize)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*CacheMeta
	for rows.Next() {
		c := &CacheMeta{}
		var remoteLM, createdTime, lastAccessed int64
		if err = rows.Scan(
			&c.Name, &c.EntityID, &c.Type, &c.Size, &c.Partial, &c.Preserved,
			&remoteLM, &createdTime, &lastAccessed,
		); err != nil {
			return nil, err
		}
		c.RemoteLastModified = time.Unix(remoteLM, 0)
		c.CreatedTime = time.Unix(createdTime, 0)
		c.LastAccessed = time.Unix(lastAccessed, 0)

		records = append(records, c)
	}

	return records, nil
}

func ListIDsByType(fileType string) ([]string, error) {
	rows, err := cacheMetaTable.Query(listSortedIDsByType, fileType)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func SummarizeCacheSize() (map[string]int64, error) {
	rows, err := cacheMetaTable.Query(sumOfSizeGroupedByType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var fileType string
		var size int64

		if err = rows.Scan(&fileType, &size); err != nil {
			return nil, err
		}

		result[fileType] = size
	}

	return result, nil
}

func SubscribeMetaTableChange() *utils.EventSubscriber[MetaChange] {
	return metaTableEm.SubscribeEvent()
}

func SubscribePreservedListChange() *utils.EventSubscriber[MetaChange] {
	return preservedListEm.SubscribeEvent()
}
