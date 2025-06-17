package icons

import (
	"embed"
	"fyne.io/fyne/v2"
	"io"
	"sync"
)

//go:embed *.svg
var FS embed.FS
var resources = map[string]fyne.Resource{}
var resourcesMutex sync.Mutex

func GetIcon(name string) fyne.Resource {
	resourcesMutex.Lock()
	if r, ok := resources[name]; ok {
		resourcesMutex.Unlock()
		return r
	}
	resourcesMutex.Unlock()
	return loadIcon(name)
}

func loadIcon(name string) fyne.Resource {
	resourcesMutex.Lock()
	defer resourcesMutex.Unlock()

	f, err := FS.Open(name + ".svg")
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
