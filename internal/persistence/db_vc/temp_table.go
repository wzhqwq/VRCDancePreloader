package db_vc

import (
	"database/sql"
	"fmt"
)

type TempTable struct {
	Table

	original *Table

	tx *sql.Tx
}

func (t *Table) ToTempTable(tx *sql.Tx) *TempTable {
	return &TempTable{
		Table: Table{
			name:    t.name + "_temp",
			columns: t.columns,
		},
		original: t,
		tx:       tx,
	}
}

func (t *TempTable) InitStructure() error {
	_, err := t.tx.Exec(t.toCreationDDL())
	return err
}

func (t *TempTable) InitIndices() error {
	for _, c := range t.columns {
		err := c.syncIndexingState(t.tx)
		if err != nil {
			logger.ErrorLnf("Failed to index %s.%s: %v", t.name, c.name, err)
		}
	}
	return nil
}

func (t *TempTable) ReplaceOriginal() error {
	if _, err := t.tx.Exec(t.original.toRemovalDDL()); err != nil {
		return err
	}
	renameDDL := fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", t.name, t.original.name)
	if _, err := t.tx.Exec(renameDDL); err != nil {
		return err
	}

	return nil
}
