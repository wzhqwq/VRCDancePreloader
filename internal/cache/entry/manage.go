package entry

var fileFormat int
var forceExpirationCheck bool

func SetFileFormat(format int) {
	fileFormat = format
}
func SetForceExpirationCheck(b bool) {
	forceExpirationCheck = b
}
