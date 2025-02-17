package persistence

const allowListTableSQL = `
CREATE TABLE IF NOT EXISTS allow_list (
    		id TEXT PRIMARY KEY,
    		size INTEGER
);
`
