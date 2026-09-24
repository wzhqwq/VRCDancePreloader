package mixed_server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// passThroughBodyTimeout bounds how long one pass-through response may deliver
// data before it gives up.
//
// It exists so that the goroutine watching the cache entry is guaranteed to
// terminate: the entry may stay uninitialized for as long as a stalled download
// queue wants (that is the point of serving from the origin instead of waiting),
// and the client may keep the connection open just as long, so neither of the
// other two branches has to happen. When this one fires, the response is cut
// like a hand-off, so the player requests the video again — by then against an
// entry the download has initialized.
const passThroughBodyTimeout = 30 * time.Minute

var passThroughLogger = utils.NewLogger("Pass Through")

// canPassThrough reports whether this request carries everything an upstream
// connection needs.
//
// The streaming path dials the origin itself, so it has to be able to name a
// destination. A request that reached the local `/cached` or `/download`
// endpoint cannot: it carries an id and nothing else, so there is no host to
// dial. Those callers keep the old contract of a request — wait for the cache
// entry, report NotFound when it never becomes usable.
func canPassThrough(req *http.Request) bool {
	if req.URL == nil || req.Host == "" {
		return false
	}

	return req.Method == http.MethodGet || req.Method == http.MethodPost
}

// passThroughDestination is the absolute URL the request has to be replayed to.
//
// A request that came through the proxy carries an absolute URL already (the
// MITM path builds one). One read off the CONNECT path carries only its path,
// and then the host it was addressed to is what completes it — a request
// hijacked there is plain HTTP by construction.
func passThroughDestination(req *http.Request) *url.URL {
	if req.URL.IsAbs() {
		return req.URL
	}

	destination := *req.URL
	destination.Scheme = "http"
	destination.Host = req.Host

	return &destination
}

// passThroughDialAddress is the host:port the upstream connection is made to.
//
// A request read off the CONNECT path does not carry the default port, so it is
// added here; without it the dial would have no port at all.
func passThroughDialAddress(req *http.Request) string {
	host := req.Host

	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "80")
	}

	return host
}

// handlePassThrough serves the video straight from the origin until the cache
// entry becomes usable, then cuts the response short.
//
// This is the first two stages of the temporary song hand-off, in the handler
// rather than in goproxy: a request the queue is not going to serve yet must not
// wait for the queue (that is the old behavior, and it ends in a 30s timeout and
// a failure), and it must stay ours to interrupt. Returning false to let goproxy
// forward it would lose the second half of that — goproxy's own forwarding is
// not addressable, so nothing could stop it when the entry becomes ready.
//
// The cut is the hand-off: a body that ends before its own Content-Length is a
// truncation the player answers with another request, and that request is served
// from the cache.
//
// It reports whether the response was served (or deliberately abandoned) here.
// false means nothing was written and the caller must fall back.
func (s *mixedServer) handlePassThrough(w http.ResponseWriter, req *http.Request, id string, entryInitialized <-chan struct{}) bool {
	if !canPassThrough(req) {
		return false
	}

	destination := passThroughDestination(req)

	upstream, err := connectDial(req.Context(), "tcp", passThroughDialAddress(req))
	if err != nil {
		passThroughLogger.ErrorLn("Cannot reach", destination.Host, "for", id, ":", err)
		return false
	}
	defer upstream.Close()

	outbound := req.Clone(req.Context())
	outbound.URL = destination
	outbound.RequestURI = ""
	outbound.Close = true
	removeHopByHopHeaders(outbound.Header)

	if err := outbound.Write(upstream); err != nil {
		passThroughLogger.ErrorLn("Cannot send the request of", id, "to", destination.Host, ":", err)
		return false
	}

	remote := bufio.NewReader(upstream)

	resp, err := http.ReadResponse(remote, outbound)
	if err != nil {
		passThroughLogger.ErrorLn("No response for", id, "from", destination.Host, ":", err)
		return false
	}
	defer resp.Body.Close()

	// Only the header is forwarded up front; the body is streamed below. Using
	// resp.Write here would copy the whole body into the response first, and the
	// hand-off below would then only ever be observed after the video had already
	// been passed through in full.
	frameChunks := needsChunkFraming(w)

	if err := writeResponseHeader(w, resp, frameChunks); err != nil {
		passThroughLogger.ErrorLn("Cannot forward the response header of", id, ":", err)
		return false
	}

	copied := make(chan struct{})
	go func() {
		defer close(copied)

		var (
			destination io.Writer = &passThroughWriter{w: w, flusher: w}
			terminate   func()
		)

		if frameChunks {
			framed := &chunkedWriter{w: w, flusher: w}
			destination, terminate = framed, framed.terminate
		}

		// A copy that ended without an error read the origin's body to the end,
		// so a framed response is finished off properly. A hand-off leaves it
		// unterminated on purpose: that is the truncation the player reacts to.
		if _, err := io.Copy(destination, resp.Body); err == nil && terminate != nil {
			terminate()
		}
	}()

	select {
	case <-entryInitialized:
		passThroughLogger.InfoLn("Hand over", id, ": its cache entry is initialized, the response is cut so that the player requests it again")
		return true

	case <-copied:
		passThroughLogger.InfoLn("Pass through", id, "finished before its cache entry was initialized")
		return true

	case <-req.Context().Done():
		passThroughLogger.InfoLn("Pass through", id, "was abandoned by the client")
		return true

	case <-time.After(passThroughBodyTimeout):
		passThroughLogger.WarnLn("Pass through", id, "gave up after", passThroughBodyTimeout)
		return true
	}
}

// removeHopByHopHeaders drops the headers that describe the connection between
// two hops rather than the content, so that replaying the request upstream does
// not carry this hop's connection negotiations with it.
func removeHopByHopHeaders(header http.Header) {
	for _, name := range []string{
		"Proxy-Connection",
		"Proxy-Authorization",
		"Proxy-Authenticate",
		"Keep-Alive",
		"Te",
		"Trailer",
		"Upgrade",
	} {
		header.Del(name)
	}
}

// needsChunkFraming reports whether this writer puts bytes on the wire as they
// are, with no framing of its own.
//
// That is the raw writer the CONNECT path installs (WriterGivenRespWriter): it
// writes the status line and the headers verbatim and then the body bytes, so a
// response declared chunked there has to carry real chunk headers or the client
// cannot read a single byte of the body. A real http.ResponseWriter — the MITM
// path, whose response goproxy writes with resp.Write — frames a chunked
// response itself, and framing it twice would corrupt it just as badly.
func needsChunkFraming(w http.ResponseWriter) bool {
	_, raw := w.(*WriterGivenRespWriter)
	return raw
}

// writeResponseHeader writes the status line and the headers of the origin
// response, and flushes them, so that the client starts receiving at once.
//
// The length is deliberately *not* forwarded. The body below is streamed and may
// be cut short by a hand-off, and a response that promised a Content-Length it
// then does not deliver is exactly the truncation the player has to notice; a
// chunked response says "the length is unknown" honestly and, more importantly,
// lets this hop hand the first bytes over before the origin has finished — a
// declared length would make the writer hold everything back until either the
// whole body or the transfer timeout.
//
// frameChunks says the caller is about to write the chunk framing itself. When
// it is false the transfer encoding is left to the writer, which chooses chunked
// for an unknown length on its own.
func writeResponseHeader(w http.ResponseWriter, resp *http.Response, frameChunks bool) error {
	resp.Header.Del("Content-Length")

	if frameChunks {
		resp.Header.Set("Transfer-Encoding", "chunked")
	}

	header := w.Header()
	for name, values := range resp.Header {
		for _, value := range values {
			header.Add(name, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	flushResponseWriter(w)

	return nil
}

// flushResponseWriter flushes a response writer if it can be flushed. The
// CONNECT path writes straight to the client connection, which is unbuffered,
// so not being flushable is not an error.
func flushResponseWriter(w http.ResponseWriter) {
	_ = http.NewResponseController(w).Flush()
}

// passThroughWriter flushes every chunk it forwards.
//
// Without the flush the origin body would sit in the response writer's buffer,
// which defeats the point of streaming it: the player would see nothing until
// the transfer was over, and a hand-off could then only ever be observed after
// the whole video had been passed through.
type passThroughWriter struct {
	w       io.Writer
	flusher http.ResponseWriter
}

func (p *passThroughWriter) Write(data []byte) (int, error) {
	n, err := p.w.Write(data)
	if n > 0 {
		flushResponseWriter(p.flusher)
	}

	return n, err
}

// chunkedWriter frames what it forwards as HTTP chunks, for a writer that has no
// framing of its own (see needsChunkFraming).
//
// terminate closes the body with the zero-length chunk. Not calling it is how a
// hand-off ends a response: an unterminated chunked body is a truncated one, and
// that is what makes the player ask for the video again.
type chunkedWriter struct {
	w       io.Writer
	flusher http.ResponseWriter
}

func (c *chunkedWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	if _, err := fmt.Fprintf(c.w, "%x\r\n", len(data)); err != nil {
		return 0, err
	}

	n, err := c.w.Write(data)
	if err != nil {
		return n, err
	}

	if _, err := c.w.Write([]byte("\r\n")); err != nil {
		return n, err
	}

	flushResponseWriter(c.flusher)

	return n, nil
}

func (c *chunkedWriter) terminate() {
	if _, err := c.w.Write([]byte("0\r\n\r\n")); err != nil {
		return
	}

	flushResponseWriter(c.flusher)
}
