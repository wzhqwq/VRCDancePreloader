package db_vc

import (
	"database/sql"
	"fmt"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type DeprecatedColumn[T any] struct {
	data []T
}

type Column struct {
	table *Table

	name    string
	sqlType string

	primary    bool
	indexed    bool
	decorators string

	deprecated bool
	since      utils.ShortVersion
}

func (c *Column) syncIndexingState(tx ...*sql.Tx) error {
	ddl := c.toIndexDDL(c.indexed)

	var err error
	if len(tx) > 0 {
		_, err = tx[0].Exec(ddl)
	} else {
		_, err = c.table.db.Exec(ddl)
	}

	return err
}

func (c *Column) toIndexDDL(creating bool) string {
	tName := c.table.name
	if creating {
		return fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s)", tName, c.name, tName, c.name)
	}
	return fmt.Sprintf("DROP INDEX IF EXISTS idx_%s_%s", tName, c.name)
}

func (c *Column) toDDL() string {
	def := fmt.Sprintf("\t%s %s", c.name, c.sqlType)
	if c.primary {
		def = def + " PRIMARY KEY"
	}
	if c.decorators != "" {
		def = def + " " + c.decorators
	}
	return def
}

func (c *Column) SetIndexed() *Column {
	c.indexed = true
	return c
}

func (c *Column) SetPrimary() *Column {
	c.primary = true
	return c
}

func (c *Column) SetDeprecated(since utils.ShortVersion) *Column {
	c.deprecated = true
	c.since = since
	return c
}

func (c *Column) SetDecorators(decorators string) *Column {
	c.decorators = decorators
	return c
}

func NewTextId() *Column {
	return NewText("id").SetPrimary()
}

func NewIncreasingId() *Column {
	return NewInt("id").SetPrimary().SetDecorators("AUTOINCREMENT")
}

func NewText(name string) *Column {
	return &Column{
		name:    name,
		sqlType: "TEXT",
	}
}

func NewInt(name string) *Column {
	return &Column{
		name:    name,
		sqlType: "INTEGER",
	}
}

func NewBool(name string) *Column {
	return &Column{
		name:    name,
		sqlType: "BOOLEAN",
	}
}
