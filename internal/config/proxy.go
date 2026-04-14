package config

import (
	"net/url"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
)

var skipTest = false

func SetSkipTest(b bool) {
	skipTest = b
}

type ProxyConfig struct {
	Pypy  string `yaml:"pypydance-api"`
	Wanna string `yaml:"wannadance-api"`
	DuDu  string `yaml:"dudu-fitdance-api"`

	BiliBiliAPI   string `yaml:"bilibili-api"`
	BiliBiliVideo string `yaml:"bilibili-video"`

	YoutubeVideo string `yaml:"youtube-video"`
	YoutubeApi   string `yaml:"youtube-api"`
	YoutubeImage string `yaml:"youtube-image"`

	GitHubApi    string `yaml:"github-api"`
	GitHubAssets string `yaml:"github-assets"`

	ProxyControllers map[string]*ProxyTester `yaml:"-"`
}

func GetProxyConfig() *ProxyConfig {
	return &config.Proxy
}

var defaultProxyConfig = ProxyConfig{}

func (pc *ProxyConfig) Init() {
	//TODO cancel comment after implemented YouTube preloading
	pc.ProxyControllers = map[string]*ProxyTester{
		"pypydance-api":     NewProxyTester("pypydance-api", pc.Pypy),
		"wannadance-api":    NewProxyTester("wannadance-api", pc.Wanna),
		"dudu-fitdance-api": NewProxyTester("dudu-fitdance-api", pc.DuDu),
		"bilibili-api":      NewProxyTester("bilibili-api", pc.BiliBiliAPI),
		"bilibili-video":    NewProxyTester("bilibili-video", pc.BiliBiliVideo),
		"youtube-video":     NewProxyTester("youtube-video", pc.YoutubeVideo),
		"youtube-api":       NewProxyTester("youtube-api", pc.YoutubeApi),
		"youtube-image":     NewProxyTester("youtube-image", pc.YoutubeImage),
		"github-api":        NewProxyTester("github-api", pc.GitHubApi),
		"github-assets":     NewProxyTester("github-assets", pc.GitHubAssets),
	}

	requesting.InitClient(requesting.PyPyDance, pc.Pypy)
	requesting.InitClient(requesting.WannaDance, pc.Wanna)
	requesting.InitClient(requesting.DuDuFitDance, pc.DuDu)
	requesting.InitClient(requesting.BiliBiliApi, pc.BiliBiliAPI)
	requesting.InitClient(requesting.BiliBiliVideo, pc.BiliBiliVideo)
	requesting.InitClient(requesting.YouTubeVideo, pc.YoutubeVideo)
	requesting.InitClient(requesting.YouTubeImage, pc.YoutubeImage)
	requesting.InitClient(requesting.YouTubeApi, pc.YoutubeApi)
	requesting.InitClient(requesting.GitHubApi, pc.GitHubApi)
	requesting.InitClient(requesting.GitHubAssets, pc.GitHubAssets)

	if !skipTest {
		pc.ProxyControllers["pypydance-api"].Test()
		pc.ProxyControllers["wannadance-api"].Test()
		pc.ProxyControllers["dudu-fitdance-api"].Test()
		pc.ProxyControllers["bilibili-api"].Test()
		pc.ProxyControllers["bilibili-video"].Test()
		pc.ProxyControllers["youtube-video"].Test()
	}
	if config.Youtube.EnableThumbnail {
		if !skipTest {
			pc.ProxyControllers["youtube-image"].Test()
		}
	}
	if config.Youtube.EnableApi {
		if !skipTest {
			pc.ProxyControllers["youtube-api"].Test()
		}
	}
}

func (pc *ProxyConfig) Update(item, value string) error {
	_, err := url.Parse(value)
	if err != nil {
		return err
	}

	switch item {
	case "pypydance-api":
		pc.Pypy = value
		requesting.UpdateClient(requesting.PyPyDance, value)
	case "wannadance-api":
		pc.Wanna = value
		requesting.UpdateClient(requesting.WannaDance, value)
	case "dudu-fitdance-api":
		pc.DuDu = value
		requesting.UpdateClient(requesting.DuDuFitDance, value)
	case "bilibili-api":
		pc.BiliBiliAPI = value
		requesting.UpdateClient(requesting.BiliBiliApi, value)
	case "bilibili-video":
		pc.BiliBiliVideo = value
		requesting.UpdateClient(requesting.BiliBiliVideo, value)
	case "youtube-video":
		pc.YoutubeVideo = value
		requesting.UpdateClient(requesting.YouTubeVideo, value)
	case "youtube-api":
		pc.YoutubeApi = value
		requesting.UpdateClient(requesting.YouTubeApi, value)
	case "youtube-image":
		pc.YoutubeImage = value
		requesting.UpdateClient(requesting.YouTubeImage, value)
	case "github-api":
		pc.GitHubApi = value
		requesting.UpdateClient(requesting.GitHubApi, value)
	case "github-assets":
		pc.GitHubAssets = value
		requesting.UpdateClient(requesting.GitHubAssets, value)
	default:
		logger.FatalLnf("Unknown proxy item: %s", item)
	}
	SaveConfig()
	return nil
}

func (pc *ProxyConfig) Test(item string) (bool, string) {
	switch item {
	case "pypydance-api":
		return requesting.TestClient(requesting.PyPyDance)
	case "wannadance-api":
		return requesting.TestClient(requesting.WannaDance)
	case "dudu-fitdance-api":
		return requesting.TestClient(requesting.DuDuFitDance)
	case "bilibili-api":
		return requesting.TestClient(requesting.BiliBiliApi)
	case "bilibili-video":
		return requesting.TestClient(requesting.BiliBiliVideo)
	case "youtube-video":
		return requesting.TestClient(requesting.YouTubeVideo)
	case "youtube-api":
		return requesting.TestClient(requesting.YouTubeApi)
	case "youtube-image":
		return requesting.TestClient(requesting.YouTubeImage)
	case "github-api":
		return requesting.TestClient(requesting.GitHubApi)
	case "github-assets":
		return requesting.TestClient(requesting.GitHubAssets)
	default:
		logger.FatalLnf("Unknown proxy item: %s", item)
		return false, ""
	}
}

type ProxyTester struct {
	input.Tester

	Status  input.Status
	Message string
	Input   *input.InputWithTester

	Value string
	Item  string
}

func NewProxyTester(item, value string) *ProxyTester {
	return &ProxyTester{
		Status: input.StatusUnknown,
		Value:  value,
		Item:   item,
	}
}

func (t *ProxyTester) Test() {
	if t.Status == input.StatusTesting {
		return
	}
	t.Status = input.StatusTesting
	if t.Input != nil {
		t.Input.SetTestBtn(true)
	}
	go func() {
		ok, message := config.Proxy.Test(t.Item)
		if ok {
			t.Message = i18n.T("tip_connectivity_test_pass")
			t.Status = input.StatusOk
		} else {
			t.Message = message
			t.Status = input.StatusError
		}
		if t.Input != nil {
			t.Input.SetTestBtn(false)
		}
	}()
}

func (t *ProxyTester) Save(value string) error {
	err := config.Proxy.Update(t.Item, value)
	if err != nil {
		return err
	}

	t.Value = value
	t.Status = input.StatusUnknown

	if t.Input != nil {
		t.Input.SetTestBtn(false)
	}
	return nil
}

func (t *ProxyTester) GetStatus() input.Status {
	return t.Status
}

func (t *ProxyTester) GetValue() string {
	return t.Value
}

func (t *ProxyTester) GetMessage() string {
	return t.Message
}

func (t *ProxyTester) GetInput(label string) *input.InputWithTester {
	if t.Input == nil {
		t.Input = input.NewInputWithTester(t, label)
	}
	return t.Input
}

func (t *ProxyTester) TestIfNotOk() {
	if t.Status == input.StatusOk {
		return
	}
	t.Test()
}
