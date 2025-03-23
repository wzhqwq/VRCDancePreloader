package persistence

const allowListTableSQL = `
CREATE TABLE IF NOT EXISTS allow_list (
    		id TEXT PRIMARY KEY,
    		size INTEGER
);
`

type CacheFileInfo struct {
	ID   string
	Size int64

	// not in database
	IsActive bool
}
