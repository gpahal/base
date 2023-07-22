package server

type Handler interface {
	ServeHTTP(c *C) error
}

type HandlerFunc func(c *C) error

func (f HandlerFunc) ServeHTTP(c *C) error {
	return f(c)
}
