package live

import (
	"bytes"
	"image/jpeg"
	"net/http"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/images/thumbnails"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
)

func (s *LiveServer) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := third_party_api.GetThumbnailByInternalID(id).Get()
	i := thumbnails.GetThumbnailImage(id, url)
	if i == nil {
		w.WriteHeader(http.StatusNotFound)
	}

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, i, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, id+".jpg", time.Now(), bytes.NewReader(buf.Bytes()))
}
