package server

import (
	"net/http"

	"github.com/go-chi/chi"
)

type R struct {
	app *App
	router chi.Router
	errHandler ErrorHandler
	middlewares []Middleware
}

func (app *App) NewRouter() *R {
	r := &R{app: app, router: chi.NewRouter()}
	r.router.NotFound(r.createErrorHandler(ErrNotFound))
	r.router.MethodNotAllowed(r.createErrorHandler(ErrMethodNotAllowed))
	return r
}

func (r *R) SetErrorHandler(errHandler ErrorHandler) {
	r.errHandler = errHandler
}

func (r *R) Use(middlewares ...Middleware) {
	if len(middlewares) == 0 {
		return
	}
	if r.middlewares == nil {
		r.middlewares = make([]Middleware, 0, len(middlewares))
	}
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *R) SubRouter(pattern string, sr *R) {
	r.router.Mount(pattern, sr.router)
}

func (r *R) Any(pattern string, h Handler) {
	r.router.HandleFunc(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Connect(pattern string, h Handler) {
	r.router.Connect(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Delete(pattern string, h Handler) {
	r.router.Delete(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Get(pattern string, h Handler) {
	r.router.Get(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Head(pattern string, h Handler) {
	r.router.Head(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Options(pattern string, h Handler) {
	r.router.Options(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Patch(pattern string, h Handler) {
	r.router.Patch(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Post(pattern string, h Handler) {
	r.router.Post(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Put(pattern string, h Handler) {
	r.router.Put(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) Trace(pattern string, h Handler) {
	r.router.Trace(pattern, r.createHTTPHandlerFunc(h))
}

func (r *R) createErrorHandler(err error) http.HandlerFunc {
	return r.createHTTPHandlerFunc(HandlerFunc(func(c *C) error {
		return err
	}))
}

func (r *R) createHTTPHandlerFunc(h Handler) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		c := r.app.newContext(res, req)
		if len(r.middlewares) > 0 {
			for _, m := range r.middlewares {
				h = m.Apply(h)
			}
		}
		err := h.ServeHTTP(c)
		if err != nil {
			handleError(r.errHandler, c, err)
		} else if !c.committed() {
			handleError(r.errHandler, c, NewStatusCodeHTTPError(http.StatusInternalServerError))
		}
	}
}
