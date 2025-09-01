package live

import (
	"embed"
	"io/fs"
	"path"
)

//go:embed web/index.html web/assets/*
var static embed.FS

type staticFS struct {
}

func (f staticFS) Open(name string) (fs.File, error) {
	return static.Open(path.Join("web", name))
}
