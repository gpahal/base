package log

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// NewDevelopment returns a new logger for development environments that logs
// to stderr in a human-friendly format.
func NewDevelopment() zerolog.Logger {
	log.Error()
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	return zerolog.New(output).With().Timestamp().Logger()
}

// NewProduction returns a new logger for production environments that logs to
// stderr in JSON format.
func NewProduction() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	return zerolog.New(os.Stderr).With().Timestamp().Logger()
}
