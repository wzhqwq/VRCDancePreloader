package favorite

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	widget.BaseWidget

	Entry *persistence.FavoriteEntry

	TitleWidget  *widgets.EllipseText
	FavoriteBtn  *widgets.FavoriteBtn
	IDWidget     *canvas.Text
	SkillRate    *widgets.Rate
	LikeRate     *widgets.Rate
	SyncToPypyCb *widget.Check
	Thumbnail    *widgets.Thumbnail

	Separator *widget.Separator
}

func NewItemGui(entry *persistence.FavoriteEntry) *ItemGui {
	title := widgets.NewEllipseText(entry.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	id := canvas.NewText(entry.ID, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	ig := &ItemGui{
		TitleWidget:  title,
		FavoriteBtn:  widgets.NewFavoriteBtn(entry.ID, entry.Title),
		IDWidget:     id,
		SkillRate:    widgets.NewRate(entry.Skill, i18n.T("label_skill_score"), "collection"),
		LikeRate:     widgets.NewRate(entry.Like, i18n.T("label_like_score"), "heart"),
		SyncToPypyCb: widget.NewCheck(i18n.T("label_sync_to_pypy"), nil),
		Thumbnail:    widgets.NewThumbnail(""),
		Separator:    widget.NewSeparator(),
	}

	ig.LikeRate.OnChanged = func(score int) {
		ig.Entry.UpdateLike(score)
	}
	ig.SkillRate.OnChanged = func(score int) {
		ig.Entry.UpdateSkill(score)
	}
	ig.SyncToPypyCb.OnChanged = func(b bool) {
		ig.Entry.UpdateSyncToPypy(b)
	}

	ig.ExtendBaseWidget(ig)

	ig.UpdateFavoriteEntry(entry)

	return ig
}

func (ig *ItemGui) UpdateFavoriteEntry(entry *persistence.FavoriteEntry) {
	ig.Entry = entry

	thumbnailUrl := ""
	if regexp.MustCompile(`^pypy_`).MatchString(entry.ID) {
		pypyId, err := strconv.Atoi(entry.ID[5:])
		if err == nil {
			thumbnailUrl = utils.GetPyPyThumbnailUrl(pypyId)
		}
	}
	ig.Thumbnail.ThumbnailURL = thumbnailUrl

	ig.TitleWidget.Text = entry.Title

	if entry.ID != ig.FavoriteBtn.ID {
		ig.FavoriteBtn.Destroy()
		ig.FavoriteBtn = widgets.NewFavoriteBtn(entry.ID, entry.Title)
	}
	ig.FavoriteBtn.SetFavorite(entry.IsFavorite)
	ig.IDWidget.Text = entry.ID
	ig.SkillRate.SetScore(entry.Skill)
	ig.LikeRate.SetScore(entry.Like)
	ig.SyncToPypyCb.Checked = entry.InPypy

	ig.Refresh()
}

func (ig *ItemGui) CreateRenderer() fyne.WidgetRenderer {
	return &ItemRenderer{
		ig: ig,
	}
}

type ItemRenderer struct {
	ig *ItemGui
}

func (r *ItemRenderer) MinSize() fyne.Size {
	minHeight := r.ig.TitleWidget.MinSize().Height + r.ig.IDWidget.MinSize().Height
	minHeight += r.ig.SkillRate.MinSize().Height + r.ig.LikeRate.MinSize().Height + r.ig.SyncToPypyCb.MinSize().Height
	return fyne.NewSize(300, minHeight)
}

func (r *ItemRenderer) Layout(size fyne.Size) {
	favWidth := r.ig.FavoriteBtn.MinSize().Width
	titleHeight := r.ig.TitleWidget.MinSize().Height

	r.ig.FavoriteBtn.Resize(fyne.NewSize(favWidth, favWidth))
	r.ig.FavoriteBtn.Move(fyne.NewPos(size.Width-favWidth, (titleHeight-r.ig.FavoriteBtn.MinSize().Height)/2))

	r.ig.TitleWidget.Resize(fyne.NewSize(size.Width-favWidth-theme.Padding(), titleHeight))
	r.ig.TitleWidget.Move(fyne.NewPos(0, 0))

	infoHeight := r.ig.SkillRate.MinSize().Height + r.ig.LikeRate.MinSize().Height + r.ig.SyncToPypyCb.MinSize().Height + r.ig.IDWidget.MinSize().Height

	imageWidth := float32(0)
	if size.Width > 320 {
		imageWidth = 160
		if size.Width < 400 {
			imageWidth = size.Width - 240
		}
		r.ig.Thumbnail.Resize(fyne.NewSize(imageWidth, infoHeight))
		r.ig.Thumbnail.Move(fyne.NewPos(0, titleHeight))
		r.ig.Thumbnail.Show()

		imageWidth += theme.Padding()
	} else {
		r.ig.Thumbnail.Hide()
	}

	r.ig.IDWidget.Resize(r.ig.IDWidget.MinSize())
	r.ig.IDWidget.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.ig.IDWidget.MinSize().Height

	r.ig.LikeRate.Resize(r.ig.LikeRate.MinSize())
	r.ig.LikeRate.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.ig.LikeRate.MinSize().Height

	r.ig.SkillRate.Resize(r.ig.SkillRate.MinSize())
	r.ig.SkillRate.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.ig.SkillRate.MinSize().Height

	r.ig.SyncToPypyCb.Resize(r.ig.SyncToPypyCb.MinSize())
	r.ig.SyncToPypyCb.Move(fyne.NewPos(imageWidth, titleHeight))

	r.ig.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.ig.Separator.Move(fyne.NewPos(0, size.Height-1))
}

func (r *ItemRenderer) Refresh() {
	r.ig.TitleWidget.Refresh()
	r.ig.Thumbnail.Refresh()
}

func (r *ItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.ig.Separator,
		r.ig.TitleWidget,
		r.ig.FavoriteBtn,
		r.ig.IDWidget,
		r.ig.SkillRate,
		r.ig.LikeRate,
		r.ig.SyncToPypyCb,
		r.ig.Thumbnail,
	}
}

func (r *ItemRenderer) Destroy() {
}
