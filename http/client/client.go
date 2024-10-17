package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/gpahal/golib/retry"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/publicsuffix"
)

const (
	defaultTimeout = 30 * time.Second
)

type Client struct {
	client    *http.Client
	baseUrl   *url.URL
	header    http.Header
	retryOpts retry.Options
}

type Options struct {
	BaseUrl          *url.URL
	BaseUrlString    string
	Timeout          time.Duration
	Header           http.Header
	RetryOpts        retry.Options
	IncludeCookieJar bool
}

func New() (*Client, error) {
	return NewWithOptions(Options{})
}

func NewWithOptions(opts Options) (*Client, error) {
	baseUrl := opts.BaseUrl
	if baseUrl == nil && opts.BaseUrlString != "" {
		var err error
		baseUrl, err = url.Parse(opts.BaseUrlString)
		if err != nil {
			return nil, err
		}
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	var cookieJar http.CookieJar
	if opts.IncludeCookieJar {
		cookieJar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
		},
		Jar: cookieJar,
	}

	return &Client{client: httpClient, baseUrl: baseUrl, header: opts.Header, retryOpts: opts.RetryOpts}, nil
}

type Request struct {
	*http.Request
}

func (c Client) NewRequest(method, urlString string) (*Request, error) {
	return c.NewRequestWithContext(context.Background(), method, urlString)
}

func (c Client) NewRequestWithContext(ctx context.Context, method, urlString string) (*Request, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	fullURL := url.String()
	if c.baseUrl != nil {
		fullURL = c.baseUrl.ResolveReference(url).String()
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header = c.header
	return &Request{Request: httpReq}, nil
}

func (req Request) GetHttpRequest() *http.Request {
	return req.Request
}

func (req *Request) SetBody(body io.Reader) {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = io.NopCloser(body)
	}

	req.Body = rc
	if rc == nil {
		req.ContentLength = 0
		req.Body = http.NoBody
		req.GetBody = func() (io.ReadCloser, error) {
			return http.NoBody, nil
		}
		return
	}

	if rc != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
			buf := v.Bytes()
			req.GetBody = func() (io.ReadCloser, error) {
				r := bytes.NewReader(buf)
				return io.NopCloser(r), nil
			}
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		default:
			if body != http.NoBody {
				req.ContentLength = -1
			}
		}

		if req.ContentLength == 0 {
			req.Body = http.NoBody
			req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		}
	}
}

func (req *Request) SetBodyJson(body any) error {
	req.Header.Set("Content-Type", "application/json")
	bs, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req.SetBody(bytes.NewReader(bs))
	return nil
}

func (req *Request) SetBodyForm(data url.Values) {
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBody(strings.NewReader(data.Encode()))
}

type Response struct {
	*http.Response
}

func (resp Response) GetHttpResponse() *http.Response {
	return resp.Response
}

func (resp Response) GetBodyString() (string, error) {
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func (resp Response) BindBodyJson(v any) error {
	err := json.NewDecoder(resp.Body).Decode(v)
	if err == nil {
		return nil
	}

	if ute, ok := err.(*json.UnmarshalTypeError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
	} else if se, ok := err.(*json.SyntaxError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
	}
	return err
}

func (c Client) Do(req *Request) (*Response, error) {
	var resp *Response
	err := retry.Do(func() error {
		httpResp, err := c.client.Do(req.Request)
		if err != nil {
			return err
		}

		resp = &Response{Response: httpResp}
		return nil
	}, c.retryOpts)

	return resp, err
}
