package icons

import (
	"embed"
	"io"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed *.svg
var FS embed.FS
var resources = map[string]fyne.Resource{}
var resourcesMutex sync.RWMutex

func getRes(key string) (fyne.Resource, bool) {
	resourcesMutex.RLock()
	defer resourcesMutex.RUnlock()
	res, ok := resources[key]
	return res, ok
}
func setRes(key string, res fyne.Resource) {
	resourcesMutex.Lock()
	defer resourcesMutex.Unlock()
	resources[key] = res
}
func readFile(name string) (fyne.Resource, error) {
	f, err := FS.Open(name + ".svg")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return fyne.NewStaticResource(name, data), nil
}

func GetIcon(name string) fyne.Resource {
	if r, ok := getRes(name); ok {
		return r
	}
	return loadIcon(name)
}

func GetColoredIcon(name string, colorName fyne.ThemeColorName) fyne.Resource {
	if r, ok := getRes(name + "-" + string(colorName)); ok {
		return r
	}
	return loadColoredIcon(name, colorName)
}

func loadIcon(name string) fyne.Resource {
	r, err := readFile(name)
	if err != nil {
		return nil
	}
	setRes(name, r)

	return r
}

func loadColoredIcon(name string, colorName fyne.ThemeColorName) fyne.Resource {
	r, err := readFile(name)
	if err != nil {
		return nil
	}
	r = theme.NewColoredResource(r, colorName)
	setRes(name+"-"+string(colorName), r)

	return r
}
