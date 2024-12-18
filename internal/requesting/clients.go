package requesting

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
	"log"
	"net/http"
	"net/url"
)

var pypyClient *http.Client
var youtubeVideoClient *http.Client
var youtubeApiClient *http.Client

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
		_, err := pypyClient.Head(utils.GetPyPyVideoUrl(1))
		if err != nil {
			log.Fatalf("Cannot connect to PyPy service through provided proxy: %v", err)
		}
		return
	}
	pypyClient = &http.Client{}
	_, err := pypyClient.Head(utils.GetPyPyVideoUrl(1))
	if err != nil {
		log.Fatalf("Cannot connect to PyPy service, maybe you should configure proxy: %v", err)
	}
}

func InitYoutubeVideoClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeVideoClient = createProxyClient(proxyUrl)
		_, err := youtubeVideoClient.Head("https://www.youtube.com")
		if err != nil {
			log.Fatalf("Cannot connect to Youtube video service through provided proxy: %v", err)
		}
		return
	}
	youtubeVideoClient = &http.Client{}
	_, err := youtubeVideoClient.Head("https://www.youtube.com")
	if err != nil {
		log.Fatalf("Cannot connect to Youtube video service, maybe you should configure proxy: %v", err)
	}
}

func InitYoutubeApiClient(proxyUrl string) {
	if proxyUrl != "" {
		youtubeApiClient = createProxyClient(proxyUrl)
		_, err := youtubeApiClient.Head("https://youtube.googleapis.com/")
		if err != nil {
			log.Fatalf("Cannot connect to Youtube API service through provided proxy: %v", err)
		}
		return
	}
	youtubeApiClient = &http.Client{}
	_, err := youtubeApiClient.Head("https://www.youtube.com")
	if err != nil {
		log.Fatalf("Cannot connect to Youtube API service, maybe you should configure proxy: %v", err)
	}
}
