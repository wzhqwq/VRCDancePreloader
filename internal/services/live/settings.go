package live

import (
	"net/http"
	"regexp"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
)

var liveUaRegex = regexp.MustCompile("OBS|livehime")

func (s *Server) handleCheckLive(w http.ResponseWriter, r *http.Request) {
	ua := r.Header.Get("User-Agent")

	writeOk(w, liveUaRegex.MatchString(ua))
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	if version == "" {
		version = "1.0"
	}
	writeOk(w, s.getSetting(version))
}

func (s *Server) getSetting(webVersion string) string {
	setting, ok := s.frontSettingsByVersion[webVersion]
	if ok {
		return setting
	}

	setting = persistence.GetLiveConfig(webVersion)
	s.frontSettingsByVersion[webVersion] = setting

	return setting
}

func (s *Server) setSetting(webVersion string, setting string) {
	s.frontSettingsByVersion[webVersion] = setting
	persistence.SetLiveConfig(webVersion, setting)
}
