package requesting

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"net/http"
	"net/url"
)

var pypyClient *http.Client
var youtubeVideoClient *http.Client
var youtubeApiClient *http.Client
var youtubeImageClient *http.Client

var skipTest = false

func SetSkipTest(b bool) {
	skipTest = b
}

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

func testClient(client *http.Client, testUrl, serviceName string) {
	if skipTest {
		return
	}
	log.Printf("Testing %s client", serviceName)
	_, err := client.Head(testUrl)
	if err != nil {
		if client.Transport == nil {
			log.Printf("[Warning] Cannot connect to %s service, maybe you should configure proxy: %v", serviceName, err)
		} else {
			log.Printf("[Warning] Cannot connect to %s service through provided proxy: %v", serviceName, err)
		}
	}
}

func InitPypyClient(proxyUrl string) {
	if proxyUrl != "" {
		pypyClient = createProxyClient(proxyUrl)
	} else {
		pypyClient = &http.Client{}
	}
	testClient(pypyClient, utils.GetPyPyVideoUrl(1), "PyPyDance")
}

func InitYoutubeVideoClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeVideoClient = createProxyClient(proxyUrl)
	} else {
		youtubeVideoClient = &http.Client{}
	}
	testClient(youtubeVideoClient, utils.GetStandardYoutubeURL("qylu4Ajh6k8"), "Youtube video")
}

func InitYoutubeApiClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeApiClient = createProxyClient(proxyUrl)
	} else {
		youtubeApiClient = &http.Client{}
	}
	testClient(youtubeApiClient, "https://youtube.googleapis.com/", "Youtube API")
}

func InitYoutubeImageClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeImageClient = createProxyClient(proxyUrl)
	} else {
		youtubeImageClient = &http.Client{}
	}
	testClient(youtubeImageClient, utils.GetYoutubeMQThumbnailURL("qylu4Ajh6k8"), "Youtube thumbnail")
}

func GetPyPyClient() *http.Client {
	return pypyClient
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
