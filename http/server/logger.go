package server

import (
	"fmt"
	"io"

	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
)

func newLogger(w io.Writer) *zerolog.Logger {
	loggerStruct := zerolog.New(
		zerolog.ConsoleWriter{
			Out:         w,
			TimeFormat:  "02 Jan 06 15:04:05 MST",
			FieldsOrder: []string{"status", "method", "uri", "error", "request_id", "latency", "size"},
		},
	).
		With().
		Timestamp().
		Logger()
	return &loggerStruct
}

type gommonLogger struct {
	logger *zerolog.Logger
	w      io.Writer
	level  log.Lvl
	prefix string
}

func newGommonLogger(logger *zerolog.Logger, loggerWriter io.Writer) *gommonLogger {
	return &gommonLogger{
		logger: logger,
		w:      loggerWriter,
		level:  getGommonLevel(logger.GetLevel()),
	}
}

func (l gommonLogger) Output() io.Writer {
	return l.w
}

func (l *gommonLogger) SetOutput(w io.Writer) {
	l.w = w
	newLogger := l.logger.Output(w)
	l.logger = &newLogger
}

func (l gommonLogger) Level() log.Lvl {
	return l.level
}

func (l *gommonLogger) SetLevel(level log.Lvl) {
	l.level = level
	newLogger := l.logger.Level(getZerologLevel(level))
	l.logger = &newLogger
}

func (l gommonLogger) Prefix() string {
	return l.prefix
}

func (l *gommonLogger) SetPrefix(prefix string) {
	l.prefix = prefix
	newLogger := l.logger.With().Str("prefix", prefix).Logger()
	l.logger = &newLogger
}

func (l gommonLogger) SetHeader(header string) {
	// Unsupported
}

type GommonLoggerContextUpdate struct {
	*zerolog.Context
	l *gommonLogger
}

func (l *gommonLogger) With() *GommonLoggerContextUpdate {
	ctx := l.logger.With()
	return &GommonLoggerContextUpdate{
		Context: &ctx,
		l:       l,
	}
}

func (u *GommonLoggerContextUpdate) Update() {
	newLogger := u.Context.Logger()
	u.l.logger = &newLogger
}

func (l gommonLogger) Debug(i ...interface{}) {
	l.logger.Debug().Msg(fmt.Sprint(i...))
}

func (l gommonLogger) Debugf(format string, i ...interface{}) {
	l.logger.Debug().Msgf(format, i...)
}

func (l gommonLogger) Debugj(j log.JSON) {
	logJson(l.logger.Debug(), j)
}

func (l gommonLogger) Info(i ...interface{}) {
	l.logger.Info().Msg(fmt.Sprint(i...))
}

func (l gommonLogger) Infof(format string, i ...interface{}) {
	l.logger.Info().Msgf(format, i...)
}

func (l gommonLogger) Infoj(j log.JSON) {
	logJson(l.logger.Info(), j)
}

func (l gommonLogger) Warn(i ...interface{}) {
	l.logger.Warn().Msg(fmt.Sprint(i...))
}

func (l gommonLogger) Warnf(format string, i ...interface{}) {
	l.logger.Warn().Msgf(format, i...)
}

func (l gommonLogger) Warnj(j log.JSON) {
	logJson(l.logger.Warn(), j)
}

func (l gommonLogger) Error(i ...interface{}) {
	l.logger.Error().Msg(fmt.Sprint(i...))
}

func (l gommonLogger) Errorf(format string, i ...interface{}) {
	l.logger.Error().Msgf(format, i...)
}

func (l gommonLogger) Errorj(j log.JSON) {
	logJson(l.logger.Error(), j)
}

func (l gommonLogger) Fatal(i ...interface{}) {
	l.logger.Fatal().Msg(fmt.Sprint(i...))
}

func (l gommonLogger) Fatalf(format string, i ...interface{}) {
	l.logger.Fatal().Msgf(format, i...)
}

func (l gommonLogger) Fatalj(j log.JSON) {
	logJson(l.logger.Fatal(), j)
}

func (l gommonLogger) Panic(i ...interface{}) {
	l.logger.Panic().Msg(fmt.Sprint(i...))
}

func (l gommonLogger) Panicf(format string, i ...interface{}) {
	l.logger.Panic().Msgf(format, i...)
}

func (l gommonLogger) Panicj(j log.JSON) {
	logJson(l.logger.Panic(), j)
}

func (l gommonLogger) Print(i ...interface{}) {
	l.Info(i...)
}

func (l gommonLogger) Printf(format string, i ...interface{}) {
	l.Infof(format, i...)
}

func (l gommonLogger) Printj(j log.JSON) {
	l.Infoj(j)
}

func logJson(event *zerolog.Event, j log.JSON) {
	for k, v := range j {
		event = event.Interface(k, v)
	}

	event.Msg("")
}

func getGommonLevel(level zerolog.Level) log.Lvl {
	switch level {
	case zerolog.NoLevel, zerolog.Disabled:
		return log.OFF
	case zerolog.TraceLevel, zerolog.DebugLevel:
		return log.DEBUG
	case zerolog.InfoLevel:
		return log.INFO
	case zerolog.WarnLevel:
		return log.WARN
	case zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel:
		return log.ERROR
	default:
		return log.DEBUG
	}
}

func getZerologLevel(level log.Lvl) zerolog.Level {
	switch level {
	case log.OFF:
		return zerolog.NoLevel
	case log.DEBUG:
		return zerolog.TraceLevel
	case log.INFO:
		return zerolog.InfoLevel
	case log.WARN:
		return zerolog.WarnLevel
	case log.ERROR:
		return zerolog.ErrorLevel
	default:
		return zerolog.TraceLevel
	}
}
