package playlist

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
)

var criticalUpdateCh = make(chan struct{}, 1)

func CriticalUpdate() {
	select {
	case criticalUpdateCh <- struct{}{}:
	default:
	}
}

func keepCriticalUpdate() {
	for {
		<-criticalUpdateCh
		PrintPlaylist()
		PreloadPlaylist()
	}
}

func PrintPlaylist() {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{i18n.T("key_id"), i18n.T("key_title"), i18n.T("key_status")})
	t.AppendRows(lo.Map(currentPlaylist, func(item *PlayItem, _ int) table.Row {
		if item.ID >= 0 {
			return table.Row{item.ID, item.Title, i18n.T("status_" + string(item.Status))}
		}
		return table.Row{i18n.T("placeholder_custom_song"), item.Title, item.Status}
	}))
	t.Render()
}

func PreloadPlaylist() {
	scanned := 0
	for _, item := range currentPlaylist {
		if scanned >= maxPreload {
			break
		}
		switch item.Status {
		case constants.Playing, constants.Ended:
			continue
		case constants.Pending:
			go item.Download()
		case constants.Failed:
			item.UpdateStatus(constants.Pending)
		}
		scanned++
	}
}
