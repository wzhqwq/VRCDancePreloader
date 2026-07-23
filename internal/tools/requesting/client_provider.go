package requesting

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func createProxyClient(proxyURL string) *http.Client {
	// TODO use github.com/rapid7/go-get-proxied/proxy to get system proxy
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
	ClientChanged  ClientEvent = "changed"
	ClientShutdown ClientEvent = "shutdown"
)

var ErrClientChanged = errors.New("proxy configuration changed")
var ErrShutdown = errors.New("shutting down")

type ClientProvider struct {
	client *http.Client
	name   string

	ProxyUrl string

	tc testCase

	tester interactive.StatefulTester

	em *utils.EventManager[ClientEvent]

	wg sync.WaitGroup
}

func NewProxyProvider(proxyUrl, name string, tc testCase) *ClientProvider {
	var c *http.Client
	if proxyUrl != "" {
		c = createProxyClient(proxyUrl)
	} else {
		c = &http.Client{}
	}

	p := &ClientProvider{
		client: c,
		name:   name,

		ProxyUrl: proxyUrl,

		tc: tc,

		em: utils.NewEventManager[ClientEvent](),
	}

	p.tester = interactive.NewTester(func() error {
		return testClient(p.client, p.name, p.tc)
	})

	return p
}

func (p *ClientProvider) SetProxy(proxyUrl string) {
	if proxyUrl != "" {
		p.client = createProxyClient(proxyUrl)
	} else {
		p.client = &http.Client{}
	}
	p.tester.Reset()
	p.em.NotifySubscribers(ClientChanged)
}

func (p *ClientProvider) AddRedirectionInterceptor() {
	if p.client.CheckRedirect != nil {
		return
	}
	p.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 10 {
			return http.ErrUseLastResponse
		}
		if req.URL.Host == "www.youtube.com" {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

func (p *ClientProvider) Tester() interactive.StatefulTester {
	return p.tester
}

func (p *ClientProvider) Client() *http.Client {
	return p.client
}

func (p *ClientProvider) Context(parent context.Context) context.Context {
	ctx, cancel := context.WithCancelCause(parent)
	p.wg.Go(func() {
		ch := p.em.SubscribeEvent()
		defer ch.Close()

		select {
		case ev := <-ch.Channel:
			switch ev {
			case ClientChanged:
				cancel(ErrClientChanged)
			case ClientShutdown:
				cancel(ErrShutdown)
			}
		case <-ctx.Done():
		}
	})

	return ctx
}

func (p *ClientProvider) SubscribeChange() *utils.EventSubscriber[ClientEvent] {
	return p.em.SubscribeEvent()
}

func (p *ClientProvider) NewGetRequest(url string, ctx context.Context) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, "GET", url, nil)
}

//func (p *ClientProvider) Get(url string) (*http.Response, error) {
//	req, err := p.NewGetRequest(url, context.Background())
//	if err != nil {
//		return nil, err
//	}
//
//	SetupHeader(req, url)
//
//	return p.Do(req)
//}

func (p *ClientProvider) Do(req *http.Request) (*http.Response, error) {
	p.AddRedirectionInterceptor()
	resp, err := p.client.Do(req)
	ctx := req.Context()
	if err != nil {
		if cause := context.Cause(ctx); errors.Is(err, context.Canceled) && cause != nil {
			// canceled by p.Context
			return nil, cause
		}
		return nil, err
	}

	resp.Body = utils.NewBodyWithContext(ctx, resp.Body)
	return resp, nil
}

func (p *ClientProvider) NewBoundGetRequest(url string, parent context.Context) (*http.Request, error) {
	return http.NewRequestWithContext(p.Context(parent), "GET", url, nil)
}

func (p *ClientProvider) BoundGet(url string) (*http.Response, error) {
	req, err := p.NewGetRequest(url, p.Context(context.Background()))
	if err != nil {
		return nil, err
	}

	SetupHeader(req, url)

	return p.Do(req)
}

func (p *ClientProvider) Shutdown() {
	p.em.NotifySubscribers(ClientShutdown)
	p.wg.Wait()
}
