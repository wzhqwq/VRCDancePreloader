package hijack

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type DeferredHeaderWriter struct {
	header        http.Header
	headerWritten bool
	statusCode    int

	headerMutex sync.Mutex

	handleWrittenHeader func(header http.Header, statusCode int)
}

func (rw *DeferredHeaderWriter) Header() http.Header {
	return rw.header
}
func (rw *DeferredHeaderWriter) WriteHeader(statusCode int) {
	rw.headerMutex.Lock()
	defer rw.headerMutex.Unlock()

	if rw.headerWritten {
		return
	}

	rw.statusCode = statusCode
	rw.handleWrittenHeader(rw.header, statusCode)
	rw.headerWritten = true
}
func (rw *DeferredHeaderWriter) BeforeWrite() {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
}

func ConstructDeferredHeaderWriter(handler func(header http.Header, statusCode int)) DeferredHeaderWriter {
	return DeferredHeaderWriter{
		header: make(http.Header),

		handleWrittenHeader: handler,
	}
}

type WriterGivenRespWriter struct {
	DeferredHeaderWriter

	writer io.Writer
}

func (rw *WriterGivenRespWriter) Write(data []byte) (int, error) {
	rw.BeforeWrite()
	return rw.writer.Write(data)
}

func NewWriterGivenRespWriter(writer io.Writer) *WriterGivenRespWriter {
	return &WriterGivenRespWriter{
		DeferredHeaderWriter: ConstructDeferredHeaderWriter(func(header http.Header, statusCode int) {
			writer.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))))

			for k, v := range header {
				for _, vv := range v {
					writer.Write([]byte(fmt.Sprintf("%s: %s\r\n", k, vv)))
				}
			}
			writer.Write([]byte("\r\n"))
		}),
	}
}

type DeferredRespWriter struct {
	DeferredHeaderWriter

	pr *io.PipeReader
	pw *io.PipeWriter

	resp *http.Response

	closeOnce sync.Once
}

func (rw *DeferredRespWriter) Write(data []byte) (int, error) {
	rw.BeforeWrite()
	return rw.pw.Write(data)
}

func (rw *DeferredRespWriter) CloseWriter() {
	rw.closeOnce.Do(func() {
		_ = rw.pw.Close()
	})
}

func NewDeferredRespWriter(req *http.Request) (*DeferredRespWriter, chan *http.Response) {
	pr, pw := io.Pipe()

	resp := &http.Response{
		Request:       req,
		Body:          pr,
		ContentLength: -1,
		ProtoMajor:    1,
		ProtoMinor:    1,
	}
	respCh := make(chan *http.Response, 1)

	rw := &DeferredRespWriter{
		DeferredHeaderWriter: ConstructDeferredHeaderWriter(func(header http.Header, statusCode int) {
			resp.Status = fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
			resp.StatusCode = statusCode
			resp.Header = header
			// resp.ContentLength
			respCh <- resp
		}),

		pr: pr,
		pw: pw,

		resp: resp,
	}

	return rw, respCh
}
