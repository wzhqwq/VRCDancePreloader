package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type RespWriter struct {
	http.ResponseWriter
	writer        io.Writer
	header        http.Header
	headerWritten bool
	statusCode    int
}

func (w *RespWriter) Header() http.Header {
	return w.header
}
func (w *RespWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}
	w.writer.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))))
	w.statusCode = statusCode

	for k, v := range w.header {
		for _, vv := range v {
			w.writer.Write([]byte(fmt.Sprintf("%s: %s\r\n", k, vv)))
		}
	}
	w.writer.Write([]byte("\r\n"))
	w.headerWritten = true
}
func (w *RespWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
		w.headerWritten = true
	}
	return w.writer.Write(b)
}
func NewRespWriter(w io.Writer) *RespWriter {
	return &RespWriter{writer: w, header: make(http.Header)}
}

type StandaloneRespWriter struct {
	RespWriter
}

func (w *StandaloneRespWriter) ToResponse(req *http.Request) *http.Response {
	bodyBuf := w.writer.(*bytes.Buffer)
	//log.Println(w.header, bodyBuf.Len(), w.statusCode)
	return &http.Response{
		Request:       req,
		StatusCode:    w.statusCode,
		Header:        w.header,
		Body:          io.NopCloser(bodyBuf),
		ContentLength: int64(bodyBuf.Len()),
	}
}

func NewStandaloneRespWriter() *StandaloneRespWriter {
	return &StandaloneRespWriter{
		RespWriter: RespWriter{
			writer: bytes.NewBuffer(make([]byte, 0, 1024)),
			header: make(http.Header),
			// Do not respond to WriteHeader so that only body is written to buffer
			headerWritten: true,
			statusCode:    http.StatusOK,
		},
	}
}
