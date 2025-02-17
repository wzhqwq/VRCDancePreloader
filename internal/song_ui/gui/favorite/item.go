package favorite

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"image/color"
)

type ItemGui struct {
	Card *fyne.Container
}

func NewItemGui(entry *persistence.FavoriteEntry) *ItemGui {
	// Title
	title := widgets.NewEllipseText(entry.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Favorite button
	favoriteBtn := widgets.NewFavoriteBtn(entry.ID, entry.Title)

	titleBar := container.NewBorder(nil, nil, nil, favoriteBtn, title)

	// ID
	id := canvas.NewText(entry.ID, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	cardContent := container.NewPadded(
		container.NewVBox(
			titleBar,
		),
	)
	cardBackground := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))
	cardBackground.CornerRadius = theme.Padding() * 2
	cardBackground.StrokeWidth = 2
	cardBackground.StrokeColor = theme.Color(theme.ColorNameSeparator)
	card := container.NewStack(cardBackground, container.NewPadded(cardContent))

	ig := ItemGui{
		Card: card,
	}

	return &ig
}
