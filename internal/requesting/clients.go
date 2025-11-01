package requesting

import (
	"log"
	"net/http"
	"net/url"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var pypyClient *http.Client
var wannaClient *http.Client
var biliClient *http.Client

var youtubeVideoClient *http.Client
var youtubeApiClient *http.Client
var youtubeImageClient *http.Client

func createProxyClient(proxyURL string) *http.Client {
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		log.Fatalf("Error parsing proxy URL: %v", err)
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}
}

func InitPypyClient(proxyUrl string) {
	if proxyUrl != "" {
		pypyClient = createProxyClient(proxyUrl)
	} else {
		pypyClient = &http.Client{}
	}
}
func TestPypyClient() (bool, string) {
	return testClient(pypyClient, "PyPyDance", videoTestCase(utils.GetPyPyVideoUrl(1)))
}

func InitWannaClient(proxyUrl string) {
	if proxyUrl != "" {
		wannaClient = createProxyClient(proxyUrl)
	} else {
		wannaClient = &http.Client{}
	}
}
func TestWannaClient() (bool, string) {
	return testClient(wannaClient, "WannaDance", videoTestCase(utils.GetWannaVideoUrl(1)))
}

func InitBiliClient(proxyUrl string) {
	if proxyUrl != "" {
		biliClient = createProxyClient(proxyUrl)
	} else {
		biliClient = &http.Client{}
	}
}
func TestBiliClient() (bool, string) {
	return testClient(biliClient, "BiliBili api", anonymousTestCase(utils.GetBiliVideoInfoURL("BV17g7XzME13")))
}

func InitYoutubeVideoClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeVideoClient = createProxyClient(proxyUrl)
	} else {
		youtubeVideoClient = &http.Client{}
	}
}
func TestYoutubeVideoClient() (bool, string) {
	return testClient(youtubeVideoClient, "Youtube video", anonymousTestCase(utils.GetStandardYoutubeURL("qylu4Ajh6k8")))
}

func InitYoutubeApiClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeApiClient = createProxyClient(proxyUrl)
	} else {
		youtubeApiClient = &http.Client{}
	}
}
func TestYoutubeApiClient() (bool, string) {
	return testClient(youtubeApiClient, "Youtube API", authenticatedTestCase("https://www.googleapis.com/youtube/v3/videos"))
}

func InitYoutubeImageClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeImageClient = createProxyClient(proxyUrl)
	} else {
		youtubeImageClient = &http.Client{}
	}
}
func TestYoutubeImageClient() (bool, string) {
	return testClient(youtubeImageClient, "Youtube thumbnail", anonymousTestCase(utils.GetYoutubeMQThumbnailURL("qylu4Ajh6k8")))
}

func GetPyPyClient() *http.Client {
	return pypyClient
}
func GetWannaClient() *http.Client {
	return wannaClient
}
func GetBiliClient() *http.Client {
	return biliClient
}
func GetYoutubeVideoClient() *http.Client {
	return youtubeVideoClient
}
func GetYoutubeApiClient() *http.Client {
	return youtubeApiClient
}
func GetYoutubeImageClient() *http.Client {
	return youtubeImageClient
}
