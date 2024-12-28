package window_app

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type cTheme struct{}

func (m cTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameSuccess {
		return color.RGBA{40, 206, 120, 255}
	}
	if name == theme.ColorNamePrimary {
		return color.RGBA{200, 30, 169, 255}
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (m cTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m cTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m cTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
