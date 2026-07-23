package live

import (
	"bytes"
	"errors"
	"image/jpeg"
	"net/http"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (s *Server) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	provider := third_parties.GetProviderById(id)
	if provider == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	handle := provider.Thumbnail(id)
	defer handle.Release()

	i, err := handle.BlockedGet(r.Context())
	if errors.Is(err, interactive.ErrUnrecoverable) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		s.svc.L().ErrorLn("Error getting thumbnail:", err)
		w.WriteHeader(http.StatusNotFound)
	}

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, i, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, id+".jpg", time.Now(), bytes.NewReader(buf.Bytes()))
}
