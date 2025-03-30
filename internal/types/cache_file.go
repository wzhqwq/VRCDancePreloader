package types

type CacheFileInfo struct {
	ID   string
	Size int64

	// not in database
	IsActive  bool
	IsPartial bool
}
