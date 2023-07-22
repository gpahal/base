package server

type Middleware interface {
	Apply(h Handler) Handler
}

type MiddlewareFunc func(h Handler) Handler

func (m MiddlewareFunc) Apply(h Handler) Handler {
	return m(h)
}
