package live

import (
	"net/http"
	"regexp"
)

var liveUaRegex = regexp.MustCompile("OBS|livehime")

type liveCheckResult struct {
	IsLive bool `json:"is_live"`
}

func (s *LiveServer) handleCheckLive(w http.ResponseWriter, r *http.Request) {
	ua := r.Header.Get("User-Agent")

	writeOk(w, &liveCheckResult{
		IsLive: liveUaRegex.MatchString(ua),
	})
}
