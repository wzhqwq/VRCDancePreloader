package db_vc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var tableNames = map[string]struct{}{}

func getTableNames(db *sql.DB) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'sql_%'")
	if err != nil {
		log.Fatalf("Failed to query tables: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		tableNames[tableName] = struct{}{}
	}
}

type Table struct {
	db *sql.DB

	name string

	columns map[string]*Column

	deprecated bool
	since      utils.ShortVersion
}

func (t *Table) toCreationDDL() string {
	cDefs := make([]string, 0, len(t.columns))

	for i := range t.columns {
		cDefs = append(cDefs, t.columns[i].toDDL())
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", t.name, strings.Join(cDefs, ",\n"))
}

func (t *Table) toRemovalDDL() string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", t.name)
}

func (t *Table) addColumn(c *Column) error {
	ddl := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", t.name, c.toDDL())
	_, err := t.db.Exec(ddl)
	return err
}

func (t *Table) getColumnsInDB() []*sql.ColumnType {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", t.name)
	rows, err := t.db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	columns, err := rows.ColumnTypes()
	if err != nil {
		log.Fatal(err)
	}

	return columns
}

var ErrUpgradeNeeded = errors.New("upgrade needed")
var ErrNotInitialized = errors.New("database is not initialized")
var ErrMismatchedPlaceholders = errors.New("the number of placeholders mismatches")

func (t *Table) Init(db *sql.DB, upgrade bool) error {
	t.db = db
	if _, ok := tableNames[t.name]; ok {
		localColumns := t.getColumnsInDB()
		for _, c := range t.columns {
			sc, ok := lo.Find(localColumns, func(item *sql.ColumnType) bool {
				return item.Name() == c.name
			})
			if !ok {
				if !upgrade {
					return ErrUpgradeNeeded
				}
				err := t.addColumn(c)
				if err != nil {
					return err
				}
			} else if sc.DatabaseTypeName() != c.sqlType {
				return fmt.Errorf("mismatched column type: %s (%s-%s)", c.name, c.sqlType, sc.DatabaseTypeName())
			}
		}
	} else {
		if !upgrade {
			return ErrUpgradeNeeded
		}
		_, err := db.Exec(t.toCreationDDL())
		if err != nil {
			return err
		}
	}
	if upgrade {
		for _, c := range t.columns {
			err := c.syncIndexingState()
			if err != nil {
				logger.ErrorLnf("Failed to index %s.%s: %v", t.name, c.name, err)
			}
		}
	}

	return nil
}

func (t *Table) Exec(query string, args ...any) (sql.Result, error) {
	return t.ExecContext(context.Background(), query, args...)
}

func (t *Table) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if t.db == nil {
		panic(ErrNotInitialized)
	}
	if strings.Count(query, "?") != len(args) {
		panic(ErrMismatchedPlaceholders)
	}
	return t.db.ExecContext(contextOrBackground(ctx), query, args...)
}

func (t *Table) Query(query string, args ...any) (*sql.Rows, error) {
	return t.QueryContext(context.Background(), query, args...)
}

func (t *Table) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if t.db == nil {
		panic(ErrNotInitialized)
	}
	if strings.Count(query, "?") != len(args) {
		panic(ErrMismatchedPlaceholders)
	}
	return t.db.QueryContext(contextOrBackground(ctx), query, args...)
}

func (t *Table) QueryRow(query string, args ...any) *sql.Row {
	return t.QueryRowContext(context.Background(), query, args...)
}

func (t *Table) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if t.db == nil {
		panic(ErrNotInitialized)
	}
	if strings.Count(query, "?") != len(args) {
		panic(ErrMismatchedPlaceholders)
	}
	return t.db.QueryRowContext(contextOrBackground(ctx), query, args...)
}

func (t *Table) Tx() (*sql.Tx, error) {
	return t.TxContext(context.Background())
}

func (t *Table) TxContext(ctx context.Context) (*sql.Tx, error) {
	if t.db == nil {
		panic(ErrNotInitialized)
	}
	return t.db.BeginTx(contextOrBackground(ctx), nil)
}

func (t *Table) Prepare(query string) (*sql.Stmt, error) {
	return t.PrepareContext(context.Background(), query)
}

func (t *Table) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if t.db == nil {
		panic(ErrNotInitialized)
	}
	return t.db.PrepareContext(contextOrBackground(ctx), query)
}

func (t *Table) Select(columns ...string) *QuickSelect {
	return newSelect(t, columns)
}

func (t *Table) Insert(columns ...string) *QuickInsert {
	return newInsert(t, columns)
}

func (t *Table) InsertOrReplace(columns ...string) *QuickInsert {
	i := newInsert(t, columns)
	i.prefix = strings.Replace(i.prefix, "INSERT", "INSERT OR REPLACE", 1)
	return i
}

func (t *Table) Update() *QuickUpdate {
	return newUpdate(t)
}

func (t *Table) Delete() *QuickDelete {
	return newDelete(t)
}

func (t *Table) DefColumn(column *Column) *Table {
	t.columns[column.name] = column
	column.table = t
	return t
}

func (t *Table) DefColumns(columns ...*Column) *Table {
	for _, c := range columns {
		t.columns[c.name] = c
		c.table = t
	}
	return t
}

func (t *Table) SetDeprecated(since utils.ShortVersion) *Table {
	t.deprecated = true
	t.since = since
	return t
}

func (t *Table) FindPrimaryKey() *Column {
	for _, c := range t.columns {
		if c.primary {
			return c
		}
	}
	return nil
}

func DefTable(name string) *Table {
	return &Table{
		name:    name,
		columns: map[string]*Column{},
	}
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
