package persistence

import (
	"database/sql"
	"errors"
)

func AddCacheMetaIfNotExists(entityID, fileType string, constructor func() *CacheMeta) *CacheMeta {
	tx, err := scheduleTable.Tx()
	if err != nil {
		logger.ErrorLn("Failed to record meta:", err)
		return constructor()
	}
	defer tx.Rollback()

	meta, ok := GetCacheMeta(entityID, fileType, tx)
	if ok {
		return meta
	}

	meta = constructor()

	err = AddCacheMeta(meta, tx)
	if err != nil {
		logger.ErrorLn("Failed to record meta:", err)
	}

	tx.Commit()

	return meta
}

func RemoveCacheMetaIfExists(entityID, fileType string) {
	tx, err := scheduleTable.Tx()
	if err != nil {
		logger.ErrorLn("Failed to record meta:", err)
		return
	}
	defer tx.Rollback()

	meta, ok := GetCacheMeta(entityID, fileType, tx)
	if ok {
		meta.Delete(tx)
	}

	tx.Commit()
}

type CacheSyncTx struct {
	SealedTx
}

func BeginCacheSyncTx() (*CacheSyncTx, error) {
	tx, err := cacheMetaTable.Tx()
	if err != nil {
		return nil, err
	}

	return &CacheSyncTx{
		SealedTx: SealedTx{tx: tx},
	}, nil
}

func (t *CacheSyncTx) ListIDsByType(fileType string) ([]string, error) {
	rows, err := t.tx.Query(listSortedIDsByType, fileType)

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

func (t *CacheSyncTx) Rebuild(records []*CacheMeta) error {
	temp := cacheMetaTable.ToTempTable(t.tx)
	addMetaTemp := temp.Insert(cacheMetaColumns...).Build()

	err := temp.InitStructure()
	if err != nil {
		return err
	}

	insertStmt, err := t.tx.Prepare(addMetaTemp)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for _, c := range records {
		_, err := insertStmt.Exec(
			c.Name, c.EntityID, c.Type, c.Size, c.Partial, c.Preserved,
			c.RemoteLastModified.Unix(), c.CreatedTime.Unix(), c.LastAccessed.Unix(),
		)
		if err != nil {
			return err
		}
	}

	err = temp.InitIndices()
	if err != nil {
		return err
	}
	err = temp.ReplaceOriginal()
	if err != nil {
		return err
	}

	logger.InfoLn("Added", len(records), "cache records")

	return t.finish()
}

func (t *CacheSyncTx) Diff(insert []*CacheMeta, delete []string) error {
	if len(insert) > 0 {
		insertStmt, err := t.tx.Prepare(addMeta)
		if err != nil {
			return err
		}
		defer insertStmt.Close()

		for _, c := range insert {
			_, err := insertStmt.Exec(
				c.Name, c.EntityID, c.Type, c.Size, c.Partial, c.Preserved,
				c.RemoteLastModified.Unix(), c.CreatedTime.Unix(), c.LastAccessed.Unix(),
			)
			if err != nil {
				return err
			}
		}

		logger.InfoLn("Added", len(insert), "cache records")
	}

	if len(delete) > 0 {
		deleteStmt, err := t.tx.Prepare(deleteMeta)
		if err != nil {
			return err
		}
		defer deleteStmt.Close()

		for _, name := range delete {
			_, err := deleteStmt.Exec(name)
			if err != nil {
				return err
			}
		}

		logger.InfoLn("Deleted", len(delete), "cache records")
	}

	return t.finish()
}

type CacheCleanupTx struct {
	SealedTx
}

func BeginCacheCleanupTx() (*CacheCleanupTx, error) {
	tx, err := cacheMetaTable.Tx()
	if err != nil {
		return nil, err
	}

	return &CacheCleanupTx{
		SealedTx: SealedTx{tx: tx},
	}, nil
}

type CleanupCandidate struct {
	ID   string
	Size int64
}

func (t *CacheCleanupTx) Summarize(fileType string) (int64, error) {
	row := t.tx.QueryRow(sumOfSizeByType, fileType)

	var size int64
	if err := row.Scan(&size); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		return 0, nil
	}

	return size, nil
}

func (t *CacheCleanupTx) ListCandidates(fileType string) ([]CleanupCandidate, error) {
	rows, err := t.tx.Query(listCleanupCandidates, fileType)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []CleanupCandidate
	for rows.Next() {
		var c CleanupCandidate
		if err = rows.Scan(&c.ID, &c.Size); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}

	return candidates, nil
}

func (t *CacheCleanupTx) MarkRemoved(entityID, fileType string) {
	meta, ok := GetCacheMeta(entityID, fileType, t.tx)
	if ok {
		meta.Delete(t.tx)
	}
}

func (t *CacheCleanupTx) Finish() error {
	return t.finish()
}
