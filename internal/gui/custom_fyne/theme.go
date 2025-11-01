package custom_fyne

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type cTheme struct{}

var (
	colorLightOuterBackground   = color.Gray{Y: 240}
	colorLightPrimary           = color.RGBA{R: 220, G: 43, B: 114, A: 255}
	colorLightPrimaryBackground = color.RGBA{R: 186, G: 44, B: 101, A: 255}
	colorLightPrimaryGrayscale  = color.Gray{Y: 180}
	colorLightButtonHover       = color.RGBA{R: 231, G: 231, B: 231, A: 255}

	colorDarkOuterBackground   = color.Gray{Y: 0}
	colorDarkPrimary           = color.RGBA{R: 186, G: 44, B: 101, A: 255}
	colorDarkPrimaryBackground = color.RGBA{R: 147, G: 43, B: 85, A: 255}
	colorDarkPrimaryGrayscale  = color.Gray{Y: 120}
	colorDarkButtonHover       = color.RGBA{R: 53, G: 54, B: 58, A: 255}
)

const (
	ColorNameOuterBackground   fyne.ThemeColorName = "outerBackground"
	ColorNamePrimaryBackground fyne.ThemeColorName = "primaryBackground"
	ColorNamePrimaryGrayscale  fyne.ThemeColorName = "primaryGrayscale"
	ColorNameButtonHover       fyne.ThemeColorName = "buttonHover"
)

func (m cTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	//variant = theme.VariantDark
	if variant == theme.VariantLight {
		switch name {
		case theme.ColorNamePrimary:
			return colorLightPrimary
		case ColorNameOuterBackground:
			return colorLightOuterBackground
		case ColorNamePrimaryBackground:
			return colorLightPrimaryBackground
		case ColorNamePrimaryGrayscale:
			return colorLightPrimaryGrayscale
		case ColorNameButtonHover:
			return colorLightButtonHover
		}
	} else {
		switch name {
		case theme.ColorNamePrimary:
			return colorDarkPrimary
		case ColorNameOuterBackground:
			return colorDarkOuterBackground
		case ColorNamePrimaryBackground:
			return colorDarkPrimaryBackground
		case ColorNamePrimaryGrayscale:
			return colorDarkPrimaryGrayscale
		case ColorNameButtonHover:
			return colorDarkButtonHover
		}
	}
	if name == theme.ColorNameSuccess {
		return color.RGBA{40, 206, 120, 255}
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
