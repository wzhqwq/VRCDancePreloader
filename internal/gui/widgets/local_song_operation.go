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

	SkillRate *Rate
	LikeRate  *Rate
}

func NewLocalSongOperations(entry *persistence.LocalSongEntry) *LocalSongOperations {
	g := &LocalSongOperations{
		Entry: entry,

		SkillRate: NewRate(entry.Skill, i18n.T("label_skill_score"), "star"),
		LikeRate:  NewRate(entry.Like, i18n.T("label_like_score"), "heart"),
	}
	g.LikeRate.OnChanged = func(score int) {
		g.Entry.UpdateLike(score)
	}
	g.SkillRate.OnChanged = func(score int) {
		g.Entry.UpdateSkill(score)
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *LocalSongOperations) UpdateEntry(entry *persistence.LocalSongEntry) {
	g.Entry = entry
	g.SkillRate.SetScore(entry.Skill)
	g.LikeRate.SetScore(entry.Like)
}

func (g *LocalSongOperations) CreateRenderer() fyne.WidgetRenderer {
	return &LocalSongOperationsRenderer{
		g: g,
	}
}

type LocalSongOperationsRenderer struct {
	g *LocalSongOperations
}

func (r *LocalSongOperationsRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.g.SkillRate.MinSize().Width, r.g.SkillRate.MinSize().Height+r.g.LikeRate.MinSize().Height)
}

func (r *LocalSongOperationsRenderer) Layout(size fyne.Size) {
	r.g.LikeRate.Resize(r.g.LikeRate.MinSize())
	r.g.LikeRate.Move(fyne.NewPos(0, 0))

	accHeight := r.g.LikeRate.MinSize().Height

	r.g.SkillRate.Resize(r.g.SkillRate.MinSize())
	r.g.SkillRate.Move(fyne.NewPos(0, accHeight))
	//accHeight += r.g.SkillRate.MinSize().Height
}

func (r *LocalSongOperationsRenderer) Refresh() {
}

func (r *LocalSongOperationsRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.g.SkillRate,
		r.g.LikeRate,
	}
}

func (r *LocalSongOperationsRenderer) Destroy() {
}
