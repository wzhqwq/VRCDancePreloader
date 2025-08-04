package history

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

func CheckRecordContinuity(parent fyne.Window) {
	if r := persistence.GetLocalRecords().GetNearestRecord(); r != nil {
		dialog.NewCustomConfirm(
			i18n.T("message_title_continue_record"),
			i18n.T("confirm_continue_record"),
			i18n.T("reject_continue_record"),
			&widget.Label{
				Alignment: fyne.TextAlignCenter,
				Text: i18n.T("message_continue_record", goeasyi18n.Options{
					Data: map[string]any{"Time": r.StartTime.Format("15:04:05")},
				}),
				Wrapping: fyne.TextWrapWord,
			},
			func(confirmed bool) {
				persistence.PrepareHistory(confirmed)
			},
			parent,
		).Show()
	} else {
		persistence.PrepareHistory(false)
	}
}

func ConfirmDeleteRecord(id int) {
	dialog.NewCustomConfirm(
		i18n.T("message_title_delete_record"),
		i18n.T("confirm_delete_record"),
		i18n.T("reject_delete_record"),
		&widget.Label{
			Alignment: fyne.TextAlignCenter,
			Text:      i18n.T("message_delete_record"),
			Wrapping:  fyne.TextWrapWord,
		},
		func(confirmed bool) {
			if confirmed {
				persistence.GetLocalRecords().DeleteRecord(id)
			}
		},
		custom_fyne.GetParent(),
	).Show()
}

type inputWithMinWidth struct {
	widget.BaseWidget

	data binding.String
}

func (i *inputWithMinWidth) CreateRenderer() fyne.WidgetRenderer {
	input := widget.NewEntryWithData(i.data)
	input.SetPlaceHolder(i18n.T("placeholder_edit_comment"))
	input.MultiLine = true
	input.Wrapping = fyne.TextWrapWord
	input.SetMinRowsVisible(10)

	return &inputWithMinWidthRenderer{
		input: input,
		i:     i,
	}
}

type inputWithMinWidthRenderer struct {
	input *widget.Entry
	i     *inputWithMinWidth
}

func (r *inputWithMinWidthRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, r.input.MinSize().Height)
}

func (r *inputWithMinWidthRenderer) Layout(size fyne.Size) {
	r.input.Resize(size)
	r.input.Move(fyne.NewPos(0, 0))
}
func (r *inputWithMinWidthRenderer) Refresh() {
	r.input.Refresh()
	canvas.Refresh(r.i)
}
func (r *inputWithMinWidthRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.input}
}
func (r *inputWithMinWidthRenderer) Destroy() {
}

func ShowCommentEditor(original string, onSubmit func(comment string)) {
	comment := binding.NewString()
	comment.Set(original)

	input := &inputWithMinWidth{
		data: comment,
	}
	input.ExtendBaseWidget(input)

	dialog.NewCustomConfirm(
		i18n.T("message_title_edit_comment"),
		i18n.T("submit_edit_comment"),
		i18n.T("discard_edit_comment"),
		input,
		func(submitted bool) {
			if submitted {
				if c, err := comment.Get(); err == nil {
					onSubmit(c)
				}
			}
		},
		custom_fyne.GetParent(),
	).Show()
}
