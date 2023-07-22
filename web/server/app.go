package server

import (
	"io"

	"github.com/rs/zerolog"
)

// App is the top-level web application instance.
type App struct {
	logger zerolog.Logger
	templateRenderer TemplateRenderer
}

// TemplateRenderer is an interface implemented by template engines.
type TemplateRenderer interface {
	Render(c *C, w io.Writer, name string, data interface{}) error
}
