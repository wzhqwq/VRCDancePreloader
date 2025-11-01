package live

import (
	"net/http"
	"regexp"
)

var liveUaRegex = regexp.MustCompile("OBS|livehime")

func (s *Server) handleCheckLive(w http.ResponseWriter, r *http.Request) {
	ua := r.Header.Get("User-Agent")

	writeOk(w, liveUaRegex.MatchString(ua))
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	writeOk(w, GetSettings())
}
