package video_cache

var maxSize int64
var keepFavorites bool

func SetMaxSize(size int64) {
	maxSize = size
}

func GetMaxSize() int64 {
	return maxSize
}

func SetKeepFavorites(b bool) {
	keepFavorites = b
}
