package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
)

type RespWriter struct {
	writer        io.Writer
	header        http.Header
	headerWritten bool
	statusCode    int
}

func (w *RespWriter) Header() http.Header {
	return w.header
}
func (w *RespWriter) WriteHeader(statusCode int) {
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

type RespWriterNoHeaderWritten struct {
	RespWriter
}

func (w *RespWriterNoHeaderWritten) ToResponse(req *http.Request) *http.Response {
	buf := w.writer.(*bytes.Buffer)
	log.Println(w.header, buf.Len(), w.statusCode)
	return &http.Response{
		Request:       req,
		StatusCode:    w.statusCode,
		Header:        w.header,
		Body:          io.NopCloser(buf),
		ContentLength: int64(buf.Len()),
	}
}

func NewRespWriterNoHeaderWritten() *RespWriterNoHeaderWritten {
	buf := make([]byte, 0, 1024)
	w := bytes.NewBuffer(buf)
	return &RespWriterNoHeaderWritten{
		RespWriter: RespWriter{
			writer:        w,
			header:        make(http.Header),
			headerWritten: true,
			statusCode:    http.StatusOK,
		},
	}
}
