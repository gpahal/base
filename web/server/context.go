package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"

	"github.com/gpahal/golib/web"
)

const (
	defaultMemory = 32 << 20 // 32 MB
	indexPage     = "index.html"
)

// C represents the context of the current HTTP request.
type C struct {
	app *App
	rww *responseWriterWrapper
	res http.ResponseWriter
	req *http.Request

	query url.Values
}

func (app *App) newContext(res http.ResponseWriter, req *http.Request) *C {
	rww, res := wrappedResponseWriter(res)
	return &C{app: app, rww:rww, res: res, req: req}
}

func (c *C) committed() bool {
	return c.rww.committed
}

func (c *C) Logger() zerolog.Logger {
	return c.app.logger
}

func (c *C) Context() context.Context {
	return c.req.Context()
}

func (c *C) URLParam(name string) string {
	return chi.URLParam(c.req, name)
}

func (c *C) QueryString() string {
	return c.req.URL.RawQuery
}

func (c *C) QueryParam(name string) string {
	if c.query == nil {
		c.query = c.req.URL.Query()
	}
	return c.query.Get(name)
}

func (c *C) QueryParams() url.Values {
	if c.query == nil {
		c.query = c.req.URL.Query()
	}
	return c.query
}

func (c *C) FormValues(name string) ([]string, error) {
	form, err := c.FormParams()
	if err != nil {
		return nil, err
	}
	return form[name], nil
}

func (c *C) FormValue(name string) (string, error) {
	if vs, err := c.FormValues(name); err != nil {
		return "", err
	} else if len(vs) > 0 {
		return vs[0], nil
	}
	return "", nil
}

func (c *C) FormParams() (url.Values, error) {
	if c.req.Form != nil {
		return c.req.Form, nil
	}

	if strings.HasPrefix(c.req.Header.Get(web.HeaderContentType), web.MIMEMultipartForm) {
		if err := c.req.ParseMultipartForm(defaultMemory); err != nil {
			return nil, err
		}
	} else {
		if err := c.req.ParseForm(); err != nil {
			return nil, err
		}
	}
	return c.req.Form, nil
}

func (c *C) FormFile(name string) (*multipart.FileHeader, error) {
	f, fh, err := c.req.FormFile(name)
	if err != nil {
		return nil, err
	}
	_ = f.Close()

	return fh, nil
}

func (c *C) MultipartForm() (*multipart.Form, error) {
	var err error
	if c.req.MultipartForm == nil {
		err = c.req.ParseMultipartForm(defaultMemory)
	}
	return c.req.MultipartForm, err
}

func (c *C) Cookie(name string) (*http.Cookie, error) {
	return c.req.Cookie(name)
}

func (c *C) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.res, cookie)
}

func (c *C) Cookies() []*http.Cookie {
	return c.req.Cookies()
}

func (c *C) Render(statusCode int, name string, data interface{}) error {
	if c.app.templateRenderer == nil {
		return ErrRendererNotRegistered
	}

	buf := new(bytes.Buffer)
	if err := c.app.templateRenderer.Render(c, buf, name, data); err != nil {
		return err
	}
	return c.htmlBlob(statusCode, buf.Bytes())
}

func (c *C) HTML(statusCode int, html string) error {
	return c.htmlBlob(statusCode, []byte(html))
}

func (c *C) htmlBlob(statusCode int, b []byte) error {
	return c.Blob(statusCode, web.MIMETextHTMLCharsetUTF8, b)
}

func (c *C) String(statusCode int, s string) error {
	return c.Blob(statusCode, web.MIMETextPlainCharsetUTF8, []byte(s))
}

func (c *C) JSON(statusCode int, i interface{}) error {
	return c.jsonInternal(statusCode, i, "")
}

func (c *C) JSONPretty(statusCode int, i interface{}, indent string) error {
	return c.jsonInternal(statusCode, i, indent)
}

func (c *C) jsonInternal(statusCode int, i interface{}, indent string) error {
	enc := json.NewEncoder(c.res)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	c.writeContentType(web.MIMEApplicationJSONCharsetUTF8)
	c.res.WriteHeader(statusCode)
	return enc.Encode(i)
}

func (c *C) JSONBlob(statusCode int, b []byte) error {
	return c.Blob(statusCode, web.MIMEApplicationJSONCharsetUTF8, b)
}

func (c *C) Blob(statusCode int, contentType string, b []byte) error {
	c.writeContentType(contentType)
	c.res.WriteHeader(statusCode)
	_, err := c.res.Write(b)
	return err
}

func (c *C) Stream(statusCode int, contentType string, r io.Reader) error {
	c.writeContentType(contentType)
	c.res.WriteHeader(statusCode)
	_, err := io.Copy(c.res, r)
	return err
}

func (c *C) File(path string) error {
	isDir, err := c.tryFile(path)
	if err != nil || !isDir {
		return err
	}

	path = filepath.Join(path, indexPage)
	isDir, err = c.tryFile(path)
	if err != nil {
		return err
	}
	if isDir {
		return ErrNotFound
	}
	return nil
}

func (c *C) tryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, ErrNotFound
		}
		return false, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	if !fi.IsDir() {
		http.ServeContent(c.res, c.req, fi.Name(), fi.ModTime(), f)
		return false, nil
	}
	return true, nil
}

func (c *C) Attachment(file, name string) error {
	return c.ContentDisposition(file, name, "attachment")
}

func (c *C) Inline(file, name string) error {
	return c.ContentDisposition(file, name, "inline")
}

func (c *C) ContentDisposition(file, name, dispositionType string) error {
	c.res.Header().Set(web.HeaderContentDisposition, fmt.Sprintf("%s; filename=%q", dispositionType, name))
	return c.File(file)
}

func (c *C) writeContentType(value string) {
	header := c.res.Header()
	header.Set(web.HeaderContentType, value)
}
