package requesting

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

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
		Transport: newGate(&http.Transport{
			Proxy: http.ProxyURL(proxy),

			// Same as http.DefaultTransport. Without it a zero-valued
			// IdleConnTimeout means "no limit", so the pooled connection to the
			// proxy would stay open for the whole process lifetime — and a
			// replaced transport (SetProxy builds a new client) would never let
			// go of it either.
			IdleConnTimeout: 90 * time.Second,
		}),
	}
}

// closeIdleConnections releases the pooled connections of a client that is being
// replaced or shut down. Nothing else can reach them afterwards, and the
// transports this package builds are never garbage collected while a connection
// is still pooled (the connection's own goroutine holds the transport).
func closeIdleConnections(client *http.Client) {
	if client == nil {
		return
	}
	if closer, ok := client.Transport.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
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
		c = &http.Client{
			Transport: newGate(nil),
		}
	}

	p := &ClientProvider{
		client: c,
		name:   name,

		ProxyUrl: proxyUrl,

		tc: tc,

		em: utils.NewEventManager[ClientEvent](),
	}

	p.tester = interactive.NewTester(func() error {
		return testClient(p.client, p.name, p.tc, p.ProxyUrl)
	})

	return p
}

func (p *ClientProvider) SetProxy(proxyUrl string) {
	previous := p.client

	if proxyUrl != "" {
		p.client = createProxyClient(proxyUrl)
	} else {
		// Transport must never be nil: mixedTransport.RoundTrip (request.go)
		// calls it directly, and a nil interface panics.
		p.client = &http.Client{Transport: newGate(nil)}
	}

	// ProxyUrl is what yt-dlp is handed (local_executables/ytdlp.go) and what the
	// availability log describes, so a hot switch has to update it as well:
	// leaving the old value here would keep passing the previous proxy to yt-dlp.
	p.ProxyUrl = proxyUrl

	// The replaced client is unreachable now, and its transport would keep its
	// pooled connections (the ones built by createProxyClient have an idle
	// timeout, but waiting them out is not the same as dropping them).
	closeIdleConnections(previous)

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

	// The clients outlive this call (the map still points at them, and SetProxy
	// can revive them), so the pooled connections are released explicitly instead
	// of being left to an idle timeout that the process may never reach.
	closeIdleConnections(p.client)
}
