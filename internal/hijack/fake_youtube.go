package hijack

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

var youTubePathRegex = regexp.MustCompile(`/([a-zA-Z0-9_-]{11})$`)

func CheckYouTubeWebPageRequest(req *http.Request) (string, bool) {
	// for v=
	if req.URL.Path == "/watch" {
		id := req.URL.Query().Get("v")
		if id != "" {
			return id, true
		}
	}
	// for path

	if matched := youTubePathRegex.FindStringSubmatch(req.URL.Path); len(matched) > 1 {
		return matched[1], true
	}
	return "", false
}

type fakeYouTubeClient struct {
	ClientName    string `json:"clientName"`
	ClientVersion string `json:"clientVersion"`
	Hl            string `json:"hl"`
	Gl            string `json:"gl"`
}

type fakeInnerTubeContext struct {
	Client fakeYouTubeClient `json:"client"`
}
type innerTubeRequest struct {
	Context fakeInnerTubeContext `json:"context"`
	VideoId string               `json:"videoId"`
}

type youtubeConfig struct {
	Context       fakeInnerTubeContext `json:"INNERTUBE_CONTEXT"`
	ClientName    string               `json:"INNERTUBE_CLIENT_NAME"`
	ClientVersion string               `json:"INNERTUBE_CLIENT_VERSION"`
	ApiKey        string               `json:"INNERTUBE_API_KEY"`
	VisitorData   string               `json:"VISITOR_DATA"`
}

var fakeInnerTubeResponseTemplate = `{
  "responseContext": {},
  "playabilityStatus": { "status": "OK" },
  "streamingData": {
    "formats": [
      {
        "itag": 18,
        "mimeType": "video/mp4; codecs=\"avc1.42001E, mp4a.40.2\"",
        "bitrate": 300000,
        "width": 640,
        "height": 360,
        "fps": 30,
        "qualityLabel": "360p",
        "url": "https://www.youtube.com/local?id=<ID>"
      }
    ]
  },
  "videoDetails": {
    "videoId": "<ID>",
    "title": "fake title",
    "lengthSeconds": "60",
    "isLiveContent": false
  }
}`

func CheckInnerTubeApiRequest(req *http.Request) (string, bool) {
	if req.URL.Path == "/youtubei/v1/player" && req.Method == http.MethodPost {
		if id := req.URL.Query().Get("videoId"); id != "" {
			return id, true
		}

		decoder := json.NewDecoder(req.Body)
		var context innerTubeRequest
		err := decoder.Decode(&context)
		if err == nil && context.VideoId != "" {
			return context.VideoId, true
		}
	}

	return "", false
}

func CheckYouTubeLocalRequest(req *http.Request) (string, bool) {
	if req.URL.Path == "/local" {
		id := req.URL.Query().Get("id")
		if id != "" {
			return id, true
		}
	}
	return "", false
}

func BuildFakeYouTubeWebpage(id string) string {
	config := youtubeConfig{
		Context: fakeInnerTubeContext{
			Client: fakeYouTubeClient{
				ClientName:    "WEB",
				ClientVersion: "2.2026.03.03",
				Hl:            "en",
				Gl:            "US",
			},
		},
	}
	configStr, err := json.Marshal(config)
	if err != nil {
		return ""
	}

	configScript := `ytcfg.set(` + string(configStr) + `);`
	dataScript := `var ytInitialData = {"contents":{"foo":"bar"}};`
	playerScript := `var ytInitialPlayerResponse = ` + BuildFakeInnerTubeApiResponse(id)
	html := `<!DOCTYPE html><html><head><script>` + configScript + dataScript + playerScript + `</script></head><body></body></html>`

	return html
}

func BuildFakeInnerTubeApiResponse(id string) string {
	return strings.ReplaceAll(fakeInnerTubeResponseTemplate, "<ID>", id)
}

func handleYouTubeWebpage(w http.ResponseWriter, id string, wg *sync.WaitGroup) bool {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	go func() {
		defer wg.Done()

		_, err := w.Write([]byte(BuildFakeYouTubeWebpage(id)))
		if err != nil {
			panic(err)
		}
	}()
	return true
}

func handleInnerTubeApi(w http.ResponseWriter, id string, wg *sync.WaitGroup) bool {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	go func() {
		defer wg.Done()

		_, err := w.Write([]byte(BuildFakeInnerTubeApiResponse(id)))
		if err != nil {
			panic(err)
		}
	}()
	return true
}
