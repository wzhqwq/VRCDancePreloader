package persistence

import (
	"database/sql"
)

type SealedTx struct {
	tx       *sql.Tx
	finished bool
}

func (t *SealedTx) finish() error {
	err := t.tx.Commit()
	if err != nil {
		return err
	}

	t.finished = true
	return nil
}

func (t *SealedTx) Abort() {
	if !t.finished {
		t.tx.Rollback()
	}
}
