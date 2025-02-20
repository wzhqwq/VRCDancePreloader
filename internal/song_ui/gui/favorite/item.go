package favorite

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"image/color"
	"regexp"
	"strconv"
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

	titleBar := container.NewPadded(container.NewBorder(nil, nil, nil, favoriteBtn, title))

	// ID
	id := canvas.NewText(entry.ID, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	// Info
	info := container.NewVBox()

	likeRate := widgets.NewRate(entry.Like, i18n.T("label_like_score"), "heart")
	likeRate.OnChanged = func(score int) {
		entry.UpdateLike(score)
	}
	info.Add(likeRate)

	skillRate := widgets.NewRate(entry.Skill, i18n.T("label_skill_score"), "collection")
	skillRate.OnChanged = func(score int) {
		entry.UpdateSkill(score)
	}
	info.Add(skillRate)

	syncToPypyCheckbox := widget.NewCheck(i18n.T("label_sync_to_pypy"), func(b bool) {
		entry.UpdateSyncToPypy(b)
	})
	syncToPypyCheckbox.SetChecked(entry.InPypy)
	info.Add(syncToPypyCheckbox)

	//videoUrl := ""
	thumbnailUrl := ""

	if regexp.MustCompile(`^pypy_`).MatchString(entry.ID) {
		pypyId, err := strconv.Atoi(entry.ID[5:])
		if err == nil {
			thumbnailUrl = utils.GetPyPyThumbnailUrl(pypyId)
			//videoUrl = utils.GetPyPyVideoUrl(pypyId)
		}
	}

	//if videoUrl != "" {
	//	url, err := url2.Parse(videoUrl)
	//	if err == nil {
	//		info.Add(widget.NewHyperlink(videoUrl, url))
	//	}
	//}

	thumbnail := widgets.NewThumbnail(thumbnailUrl)

	cardContent := container.NewVBox(
		container.NewVBox(
			titleBar,
		),
		widgets.NewDynamicFrame(
			thumbnail,
			info,
			nil,
			nil,
			nil,
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
