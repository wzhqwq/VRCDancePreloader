package proxy

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"log"
	"net/http"
	"strconv"
	"time"
)

func handleWannaRequest(w http.ResponseWriter, req *http.Request) bool {
	if req.Host != "jd.pypy.moe" {
		return false
	}
	if req.URL.Path == "/Api/Songs/play" {
		q := req.URL.Query()
		id, err := strconv.Atoi(q.Get("id"))
		if err != nil {
			log.Println("Failed to parse id:", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return true
		}

		rangeHeader := req.Header.Get("Range")
		if rangeHeader == "" {
			log.Printf("Intercepted WannaDance video %d full request", id)
		} else {
			log.Printf("Intercepted WannaDance video %d range: %s", id, rangeHeader)
		}
		reader, err := playlist.RequestWannaSong(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			log.Println("Failed to load WannaDance video:", err)
			return true
		}
		log.Printf("Requested WannaDance video %d is available", id)

		http.ServeContent(w, req, "video.mp4", time.Now(), reader)
		return true
	}

	return false
}
