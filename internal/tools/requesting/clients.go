package requesting

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

type ClientName string

const (
	NoProxy ClientName = "NoProxy"

	PyPyDance    ClientName = "PyPyDance"
	WannaDance   ClientName = "WannaDance"
	DuDuFitDance ClientName = "DuDuFitDance"

	BiliBili ClientName = "BiliBili"

	YouTubeVideo ClientName = "YouTube video"
	YouTubeApi   ClientName = "YouTube API"
	YouTubeImage ClientName = "YouTube thumbnail"

	GitHubApi    ClientName = "GitHub API"
	GitHubAssets ClientName = "GitHub Assets"
)

var clients = map[ClientName]*ClientProvider{
	NoProxy:      NewProxyProvider("", "default", testCase{}),
	PyPyDance:    nil,
	WannaDance:   nil,
	DuDuFitDance: nil,
	BiliBili:     nil,
	YouTubeVideo: nil,
	YouTubeApi:   nil,
	YouTubeImage: nil,
	GitHubApi:    nil,
	GitHubAssets: nil,
}

var testCases = map[ClientName]testCase{
	PyPyDance:    videoTestCase(internal_id.GetPyPyVideoUrl(1)),
	WannaDance:   videoTestCase(internal_id.GetWannaVideoUrl(1)),
	DuDuFitDance: videoTestCase(internal_id.GetDuDuVideoUrl(1)),
	BiliBili:     anonymousTestCaseGet(internal_id.GetBiliVideoInfoURL("BV17g7XzME13")),
	YouTubeVideo: anonymousTestCase(internal_id.GetStandardYoutubeURL("qylu4Ajh6k8")),
	// The probe has to hit the host the API client really calls: the generated
	// google.golang.org/api youtube/v3 service uses the base path
	// "https://youtube.googleapis.com/" (youtube-gen.go:100), while
	// www.googleapis.com is a different host — a self-test passing there would not
	// prove the endpoint the API uses is reachable.
	// TestAvailabilityProbeTargetsTheRealEndpoint pins this against svc.BasePath.
	YouTubeApi:   authenticatedTestCase("https://youtube.googleapis.com/youtube/v3/videos"),
	YouTubeImage: anonymousTestCase(internal_id.GetYoutubeMQThumbnailURL("qylu4Ajh6k8")),
	GitHubApi:    anonymousTestCaseGet("https://api.github.com"),
	GitHubAssets: storageServerTestCase("https://release-assets.githubusercontent.com"),
}

func initClient(name ClientName, proxyUrl string, doTest bool) {
	p := NewProxyProvider(proxyUrl, string(name), testCases[name])
	if doTest {
		go p.tester.Test()
	}
	clients[name] = p
}

func shutdownClients() {
	for _, client := range clients {
		client.Shutdown()
	}
}

func updateClient(name ClientName, proxyUrl string) {
	clients[name].SetProxy(proxyUrl)
}

func GetClient(name ClientName) *ClientProvider {
	return clients[name]
}

func GetTester(field string) interactive.StatefulTester {
	switch field {
	case "pypydance-api":
		return clients[PyPyDance].tester
	case "wannadance-api":
		return clients[WannaDance].tester
	case "dudu-fitdance-api":
		return clients[DuDuFitDance].tester
	case "bilibili":
		return clients[BiliBili].tester
	case "youtube-video":
		return clients[YouTubeVideo].tester
	case "youtube-api":
		return clients[YouTubeApi].tester
	case "youtube-image":
		return clients[YouTubeImage].tester
	case "github-api":
		return clients[GitHubApi].tester
	case "github-assets":
		return clients[GitHubAssets].tester
	default:
		return nil
	}
}
