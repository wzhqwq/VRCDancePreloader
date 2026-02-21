package db_vs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/samber/lo"
)

var stringLiteralRegex = regexp.MustCompile(`'[^']*'|"[^"]*"`)
var identifierRegex = regexp.MustCompile("`[^`]+`|[a-zA-Z_][a-zA-Z0-9_.]*")
var keywords = map[string]bool{
	"AND": true, "OR": true, "NOT": true, "IN": true, "IS": true, "NULL": true,
	"LIKE": true, "BETWEEN": true, "EXISTS": true, "TRUE": true, "FALSE": true,
	"ASC": true, "DESC": true, "DIV": true, "MOD": true, "XOR": true,
	"COUNT": true, "SUM": true, "AVG": true, "MAX": true, "MIN": true,
	"DATE": true, "NOW": true,
}

func extractColumns(expression string) []string {
	cleaned := stringLiteralRegex.ReplaceAllString(expression, " ")
	matches := identifierRegex.FindAllString(cleaned, -1)

	columnsMap := make(map[string]struct{})
	for _, match := range matches {
		name := strings.Trim(match, "`")
		upperName := strings.ToUpper(name)

		if !keywords[upperName] && len(name) > 0 {
			if match[0] >= '0' && match[0] <= '9' {
				continue
			}
			columnsMap[name] = struct{}{}
		}
	}

	result := make([]string, 0, len(columnsMap))
	for col := range columnsMap {
		result = append(result, col)
	}

	return result
}

func checkExp(table *Table, expression string) {
	checkColumns(table, extractColumns(expression))
}
func checkColumns(table *Table, columns []string) {
	for _, name := range columns {
		c, ok := table.columns[name]
		if !ok {
			logger.FatalLn("Bad expression: undefined column", name)
		}
		if c.deprecated {
			logger.WarnLn("Usage of deprecated column", name, "- since", c.since.String())
		}
	}
}

type QuickSelect struct {
	table *Table

	prefix  string
	locator string
	sorter  string
	limiter string
}

func (q *QuickSelect) Where(expression string) *QuickSelect {
	checkExp(q.table, expression)
	q.locator = " WHERE " + expression
	return q
}

func (q *QuickSelect) Sort(name string, ascending bool) *QuickSelect {
	c, ok := q.table.columns[name]
	if !ok {
		logger.FatalLn("Bad SELECT: undefined column", name)
	}
	if c.deprecated {
		logger.WarnLn("Usage of deprecated column", name)
	}

	q.sorter = " ORDER BY " + name
	if !ascending {
		q.sorter = q.sorter + " DESC"
	}

	return q
}

func (q *QuickSelect) Paginate() *QuickSelect {
	q.limiter = " LIMIT ? OFFSET ?"
	return q
}

func (q *QuickSelect) Limit(count int) *QuickSelect {
	q.limiter = fmt.Sprintf(" LIMIT %d", count)
	return q
}

func (q *QuickSelect) Build() string {
	return q.prefix + q.locator + q.sorter + q.limiter
}

func newSelect(table *Table, columns []string) *QuickSelect {
	selector := "*"
	if len(columns) > 0 {
		selector = strings.Join(columns, ", ")
	}
	checkExp(table, selector)

	return &QuickSelect{
		table:  table,
		prefix: fmt.Sprintf("SELECT %s FROM %s", selector, table.name),
	}
}

type QuickInsert struct {
	table  *Table
	prefix string
}

func newInsert(table *Table, columns []string) *QuickInsert {
	checkColumns(table, columns)

	columnExp := strings.Join(columns, ", ")
	questions := lo.RepeatBy(len(columns), func(_ int) string {
		return "?"
	})
	valueExp := strings.Join(questions, ", ")

	return &QuickInsert{
		table:  table,
		prefix: fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table.name, columnExp, valueExp),
	}
}

func (q *QuickInsert) Build() string {
	return q.prefix
}

type QuickUpdate struct {
	table *Table

	prefix  string
	setter  string
	locator string
}

func (q *QuickUpdate) Set(expressions ...string) *QuickUpdate {
	clause := strings.Join(expressions, ", ")
	checkExp(q.table, clause)
	q.setter = " SET " + clause
	return q
}

func (q *QuickUpdate) Where(expression string) *QuickUpdate {
	checkExp(q.table, expression)
	q.locator = " WHERE " + expression
	return q
}

func (q *QuickUpdate) Build() string {
	if q.setter == "" {
		logger.WarnLn("Nonsense:", q.prefix+q.locator)
	}
	return q.prefix + q.setter + q.locator
}

func newUpdate(table *Table) *QuickUpdate {
	return &QuickUpdate{
		table:  table,
		prefix: fmt.Sprintf("UPDATE %s", table.name),
	}
}

type QuickDelete struct {
	table *Table

	prefix  string
	locator string
}

func (q *QuickDelete) Where(expression string) *QuickDelete {
	checkExp(q.table, expression)
	q.locator = " WHERE " + expression
	return q
}

func (q *QuickDelete) Build() string {
	return q.prefix + q.locator
}

func newDelete(table *Table) *QuickDelete {
	return &QuickDelete{
		table:  table,
		prefix: fmt.Sprintf("DELETE FROM %s", table.name),
	}
}
