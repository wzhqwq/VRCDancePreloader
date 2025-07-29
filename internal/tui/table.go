package tui

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

type SongTable struct {
	lastStatus map[int64]string
}

func NewSongTable() *SongTable {
	t := table.NewWriter()
	t.SetOutputMirror(nil)
	t.AppendHeader(table.Row{i18n.T("key_id"), i18n.T("key_status"), i18n.T("key_title")})

	return &SongTable{
		lastStatus: map[int64]string{},
	}
}

func (st *SongTable) Print(items []*ItemTui) {
	allTheSame := true
	statusMap := map[int64]string{}
	for _, item := range items {
		status := item.ps.GetStatusInfo().Status
		id := item.ps.ID
		statusMap[id] = status
		if lastStatus, ok := st.lastStatus[id]; !ok || lastStatus != status {
			allTheSame = false
		}
	}
	if allTheSame {
		return
	}
	st.lastStatus = statusMap

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{i18n.T("key_id"), i18n.T("key_status"), i18n.T("key_title")})
	t.AppendRows(lo.Map(items, func(item *ItemTui, _ int) table.Row {
		info := item.ps.GetInfo()
		return table.Row{
			info.ID,
			st.lastStatus[item.ps.ID],
			info.Title,
		}
	}))
	t.Render()
}
