package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

type LocalSongOperations struct {
	widget.BaseWidget

	Entry *persistence.LocalSongEntry
}

func NewLocalSongOperations(entry *persistence.LocalSongEntry) *LocalSongOperations {
	g := &LocalSongOperations{
		Entry: entry,
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *LocalSongOperations) UpdateEntry(entry *persistence.LocalSongEntry) {
	g.Entry = entry
	fyne.Do(func() {
		g.Refresh()
	})
}

func (g *LocalSongOperations) CreateRenderer() fyne.WidgetRenderer {
	likeRate := NewRate(g.Entry.Like, i18n.T("label_like_score"), "heart")
	skillRate := NewRate(g.Entry.Skill, i18n.T("label_skill_score"), "star")

	likeRate.OnChanged = func(score int) {
		g.Entry.UpdateLike(score)
	}
	skillRate.OnChanged = func(score int) {
		g.Entry.UpdateSkill(score)
	}

	return &LocalSongOperationsRenderer{
		g: g,

		SkillRate: skillRate,
		LikeRate:  likeRate,
	}
}

type LocalSongOperationsRenderer struct {
	g *LocalSongOperations

	SkillRate *Rate
	LikeRate  *Rate
}

func (r *LocalSongOperationsRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.SkillRate.MinSize().Width, r.SkillRate.MinSize().Height+r.LikeRate.MinSize().Height)
}

func (r *LocalSongOperationsRenderer) Layout(size fyne.Size) {
	r.LikeRate.Resize(r.LikeRate.MinSize())
	r.LikeRate.Move(fyne.NewPos(0, 0))

	accHeight := r.LikeRate.MinSize().Height

	r.SkillRate.Resize(r.SkillRate.MinSize())
	r.SkillRate.Move(fyne.NewPos(0, accHeight))
	//accHeight += r.g.SkillRate.MinSize().Height
}

func (r *LocalSongOperationsRenderer) Refresh() {
	r.SkillRate.SetScore(r.g.Entry.Skill)
	r.LikeRate.SetScore(r.g.Entry.Like)
}

func (r *LocalSongOperationsRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.SkillRate,
		r.LikeRate,
	}
}

func (r *LocalSongOperationsRenderer) Destroy() {
}
