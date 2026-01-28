package requesting

import (
	"context"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type ClientName string

const (
	PyPyDance    ClientName = "PyPyDance"
	WannaDance   ClientName = "WannaDance"
	DuDuFitDance ClientName = "DuDuFitDance"
	BiliBiliApi  ClientName = "BiliBili API"
	YouTubeVideo ClientName = "YouTube video"
	YouTubeApi   ClientName = "YouTube API"
	YouTubeImage ClientName = "YouTube thumbnail"
)

var clients = map[ClientName]*ClientProvider{
	PyPyDance:    nil,
	WannaDance:   nil,
	DuDuFitDance: nil,
	BiliBiliApi:  nil,
	YouTubeVideo: nil,
	YouTubeApi:   nil,
	YouTubeImage: nil,
}

var testCases = map[ClientName]testCase{
	PyPyDance:    videoTestCase(utils.GetPyPyVideoUrl(1)),
	WannaDance:   videoTestCase(utils.GetWannaVideoUrl(1)),
	DuDuFitDance: videoTestCase(utils.GetDuDuVideoUrl(1)),
	BiliBiliApi:  anonymousTestCaseGet(utils.GetBiliVideoInfoURL("BV17g7XzME13")),
	YouTubeVideo: anonymousTestCase(utils.GetStandardYoutubeURL("qylu4Ajh6k8")),
	YouTubeApi:   authenticatedTestCase("https://www.googleapis.com/youtube/v3/videos"),
	YouTubeImage: anonymousTestCase(utils.GetYoutubeMQThumbnailURL("qylu4Ajh6k8")),
}

func InitClient(name ClientName, proxyUrl string) {
	clients[name] = NewProxyProvider(proxyUrl, string(name))
}

func TestClient(name ClientName) (bool, string) {
	return clients[name].Test(testCases[name])
}

func UpdateClient(name ClientName, proxyUrl string) {
	clients[name].SetProxy(proxyUrl)
}

func GetClient(name ClientName) *ClientProvider {
	return clients[name]
}

// YouTube API

func GetYoutubeApiContext() context.Context {
	return clients[YouTubeApi].Context(context.Background())
}
