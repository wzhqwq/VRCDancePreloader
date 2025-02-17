package icons

import (
	"embed"
	"fyne.io/fyne/v2"
	"io"
)

//go:embed *.png
var FS embed.FS
var resources = map[string]fyne.Resource{}

func GetIcon(name string) fyne.Resource {
	if r, ok := resources[name]; ok {
		return r
	}
	f, err := FS.Open(name + ".png")
	if err != nil {
		return nil
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	r := fyne.NewStaticResource(name, data)
	resources[name] = r
	return r
}
