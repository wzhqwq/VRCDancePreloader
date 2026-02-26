package global_state

var isGui = false
var dbMigrationPath = ""

func RunInGui() {
	isGui = true
}

func IsInGui() bool {
	return isGui
}

func SetDbMigrationPath(path string) {
	dbMigrationPath = path
}

func GetDbMigrationPath() string {
	return dbMigrationPath
}
