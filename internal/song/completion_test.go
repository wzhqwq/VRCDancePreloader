package song

import (
	"errors"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func placeholderInfo() types.GeneralVideoInfo {
	return types.GeneralVideoInfo{Title: "DuDuFitDance 1543"}
}

func fullInfo() types.GeneralVideoInfo {
	return types.GeneralVideoInfo{
		Title:     "Some Song",
		GroupName: "Some Group",
		Duration:  3 * time.Minute,
	}
}

func snapshot(data types.GeneralVideoInfo, hasData bool, status interactive.RemoteStatus) interactive.RemoteSnapshot[types.GeneralVideoInfo] {
	return interactive.RemoteSnapshot[types.GeneralVideoInfo]{
		Data:    data,
		HasData: hasData,
		Status:  status,
	}
}

// TestInfoSnapshotMoved pins the guard that keeps completionLoop quiet: the entry
// notifies on every attempt and every phase change, and re-publishing identical
// info is what looked like duplicated output (review/09 §2.4/§8.2).
func TestInfoSnapshotMoved(t *testing.T) {
	loading := interactive.RemoteStatus{Phase: interactive.RemoteLoading}
	ready := interactive.RemoteStatus{Phase: interactive.RemoteReady}
	failed := interactive.RemoteStatus{Phase: interactive.RemoteError, Err: errors.New("nope")}
	failedPending := interactive.RemoteStatus{
		Phase:         interactive.RemoteErrorRetryPending,
		Err:           errors.New("nope"),
		RetryAttempts: "1 / 5",
		RetryAfter:    time.Now(),
	}

	placeholderLoading := snapshot(placeholderInfo(), true, loading)

	cases := []struct {
		name     string
		previous interactive.RemoteSnapshot[types.GeneralVideoInfo]
		current  interactive.RemoteSnapshot[types.GeneralVideoInfo]
		want     bool
	}{
		{"the same snapshot again", placeholderLoading, placeholderLoading, false},
		{"the same info, only the phase moved", placeholderLoading, snapshot(placeholderInfo(), true, failed), false},
		{"the same info and phase, different error text", snapshot(placeholderInfo(), true, failed), snapshot(placeholderInfo(), true, interactive.RemoteStatus{Phase: interactive.RemoteError, Err: errors.New("other")}), false},
		{"retry bookkeeping only", snapshot(placeholderInfo(), true, interactive.RemoteStatus{Phase: interactive.RemoteErrorRetryPending, Err: errors.New("nope")}), snapshot(placeholderInfo(), true, failedPending), false},
		{"the same info became valid", snapshot(fullInfo(), true, loading), snapshot(fullInfo(), true, ready), false},
		{"the queue completed the placeholder", placeholderLoading, snapshot(fullInfo(), true, loading), true},
		{"the queue completed it while the fetch was failing", placeholderLoading, snapshot(fullInfo(), true, failed), true},
		{"the info was replaced", snapshot(fullInfo(), true, loading), snapshot(types.GeneralVideoInfo{Title: "Other"}, true, loading), true},
		{"data disappeared", snapshot(fullInfo(), true, loading), snapshot(types.GeneralVideoInfo{}, false, loading), true},
	}

	for _, c := range cases {
		if got := infoSnapshotMoved(c.previous, c.current); got != c.want {
			t.Errorf("%s: infoSnapshotMoved() = %v, want %v", c.name, got, c.want)
		}
	}
}
