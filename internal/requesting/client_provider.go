package requesting

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func createProxyClient(proxyURL string) *http.Client {
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		logger.FatalLn("Error parsing proxy URL:", err)
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}
}

type ClientEvent string

const (
	ClientChanged ClientEvent = "changed"
)

var ErrClientChanged = errors.New("proxy configuration changed")

type ClientProvider struct {
	client *http.Client
	name   string

	em *utils.EventManager[ClientEvent]
}

func NewProxyProvider(proxyUrl, name string) *ClientProvider {
	var c *http.Client
	if proxyUrl != "" {
		c = createProxyClient(proxyUrl)
	} else {
		c = &http.Client{}
	}

	return &ClientProvider{
		client: c,
		name:   name,
		em:     utils.NewEventManager[ClientEvent](),
	}
}

func (p *ClientProvider) SetProxy(proxyUrl string) {
	if proxyUrl != "" {
		p.client = createProxyClient(proxyUrl)
	} else {
		p.client = &http.Client{}
	}
	p.em.NotifySubscribers(ClientChanged)
}

func (p *ClientProvider) Test(tc testCase) (bool, string) {
	return testClient(p.client, p.name, tc)
}

func (p *ClientProvider) Client() *http.Client {
	return p.client
}

func (p *ClientProvider) Context(parent context.Context) context.Context {
	ctx, cancel := context.WithCancelCause(parent)
	ch := p.em.SubscribeEvent()
	go func() {
		select {
		case <-ch.Channel:
			cancel(ErrClientChanged)
		case <-ctx.Done():
		}
	}()

	return ctx
}

func (p *ClientProvider) NewGetRequest(url string, parent context.Context) (*http.Request, error) {
	return http.NewRequestWithContext(p.Context(parent), "GET", url, nil)
}

func (p *ClientProvider) Get(url string) (*http.Response, error) {
	req, err := p.NewGetRequest(url, context.Background())
	if err != nil {
		return nil, err
	}

	return p.client.Do(req)
}

func (p *ClientProvider) Do(req *http.Request) (*http.Response, error) {
	return p.client.Do(req)
}
