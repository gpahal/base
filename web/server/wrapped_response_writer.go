package server

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// responseWriterWrapper wraps an http.ResponseWriter.
type responseWriterWrapper struct {
	writer http.ResponseWriter

	size      int64
	statusCode int
	committed  bool
}

func wrappedResponseWriter(w http.ResponseWriter) (*responseWriterWrapper, http.ResponseWriter) {
	rw := &responseWriterWrapper{writer: w}
	_, i1 := w.(http.Flusher)
	_, i2 := w.(http.Hijacker)
	_, i3 := w.(io.ReaderFrom)
	switch {
	// combination 0/7
	case !i1 && !i2 && !i3:
		return rw, struct {
			http.ResponseWriter
		}{rw}
	// combination 1/7
	case !i1 && !i2 && i3:
		return rw, struct {
			http.ResponseWriter
			io.ReaderFrom
		}{rw, rw}
	// combination 2/7
	case !i1 && i2 && !i3:
		return rw, struct {
			http.ResponseWriter
			http.Hijacker
		}{rw, rw}
	// combination 3/7
	case !i1 && i2 && i3:
		return rw, struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{rw, rw, rw}
	// combination 4/7
	case i1 && !i2 && !i3:
		return rw, struct {
			http.ResponseWriter
			http.Flusher
		}{rw, rw}
	// combination 5/7
	case i1 && !i2 && i3:
		return rw, struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{rw, rw, rw}
	// combination 6/7
	case i1 && i2 && !i3:
		return rw, struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
		}{rw, rw, rw}
	// combination 7/7
	default:
		return rw, struct {
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{rw, rw, rw, rw}
	}
}

func (ww *responseWriterWrapper) Header() http.Header {
	return ww.writer.Header()
}

func (ww *responseWriterWrapper) WriteHeader(statusCode int) {
	if ww.committed {
		return
	}

	ww.statusCode = statusCode
	ww.writer.WriteHeader(statusCode)
	ww.committed = true
}

func (ww *responseWriterWrapper) Write(b []byte) (int, error) {
	if !ww.committed {
		ww.WriteHeader(http.StatusOK)
	}

	n, err := ww.writer.Write(b)
	ww.size += int64(n)
	return n, err
}

func (ww *responseWriterWrapper) Push(target string, opts *http.PushOptions) error {
	if p, ok := ww.writer.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (ww *responseWriterWrapper) Flush() {
	ww.writer.(http.Flusher).Flush()
}

func (ww *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return ww.writer.(http.Hijacker).Hijack()
}

func (ww *responseWriterWrapper) ReadFrom(src io.Reader) (int64, error) {
	n, err := ww.writer.(io.ReaderFrom).ReadFrom(src)
	ww.size += n
	return n, err
}
