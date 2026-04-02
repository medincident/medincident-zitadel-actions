package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/rs/zerolog"
	oopszerolog "github.com/samber/oops/loggers/zerolog"
)

// OutputOption configures a single output destination.
type OutputOption func(*outputCfg)

type outputCfg struct {
	level        zerolog.Level
	pretty       bool
	noColor      bool
	timeFormat   string
	partsOrder   []string
	partsExclude []string
}

func defaultOutputCfg() outputCfg {
	return outputCfg{
		level:      zerolog.TraceLevel,
		timeFormat: time.RFC3339,
	}
}

// OutputLevel sets the minimum log level for this output.
// Entries below this level are dropped regardless of the global logger level.
func OutputLevel(level zerolog.Level) OutputOption {
	return func(c *outputCfg) { c.level = level }
}

// OutputPretty enables human-readable (ConsoleWriter) formatting.
func OutputPretty() OutputOption {
	return func(c *outputCfg) { c.pretty = true }
}

// OutputJSON enforces JSON formatting (default for file/writer outputs).
func OutputJSON() OutputOption {
	return func(c *outputCfg) { c.pretty = false }
}

// OutputNoColor disables ANSI color codes in pretty mode.
func OutputNoColor() OutputOption {
	return func(c *outputCfg) { c.noColor = true }
}

// OutputTimeFormat sets the timestamp format for this output.
func OutputTimeFormat(format string) OutputOption {
	return func(c *outputCfg) { c.timeFormat = format }
}

// OutputPartsOrder sets the display order of fields in pretty output.
// Use zerolog field name constants: zerolog.TimestampFieldName, zerolog.LevelFieldName, etc.
func OutputPartsOrder(parts ...string) OutputOption {
	return func(c *outputCfg) { c.partsOrder = parts }
}

// OutputPartsExclude hides specific fields from pretty output.
func OutputPartsExclude(parts ...string) OutputOption {
	return func(c *outputCfg) { c.partsExclude = parts }
}

func buildLevelWriter(w io.Writer, cfg outputCfg) zerolog.LevelWriter {
	var base zerolog.LevelWriter
	if cfg.pretty {
		cw := zerolog.ConsoleWriter{
			Out:        w,
			NoColor:    cfg.noColor,
			TimeFormat: cfg.timeFormat,
		}
		if len(cfg.partsOrder) > 0 {
			cw.PartsOrder = cfg.partsOrder
		}
		if len(cfg.partsExclude) > 0 {
			cw.PartsExclude = cfg.partsExclude
		}
		base = zerolog.LevelWriterAdapter{Writer: cw}
	} else {
		base = zerolog.LevelWriterAdapter{Writer: w}
	}
	// TraceLevel is the lowest level — no filtering needed.
	if cfg.level <= zerolog.TraceLevel {
		return base
	}
	return &zerolog.FilteredLevelWriter{Writer: base, Level: cfg.level}
}

type loggerCfg struct {
	outputs    []zerolog.LevelWriter
	writerSet  []io.Writer
	fileSet    []string
	closers    []io.Closer
	level      zerolog.Level
	caller     bool
	callerSkip int
	timestamp  bool
	timeFormat string
	timeUTC    bool
	ctxFields  []func(zerolog.Context) zerolog.Context
}

func (c *loggerCfg) trackWriter(w io.Writer) error {
	if slices.Contains(c.writerSet, w) {
		return fmt.Errorf("zerolog: duplicate output writer %T", w)
	}
	c.writerSet = append(c.writerSet, w)
	return nil
}

func (c *loggerCfg) trackFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if slices.Contains(c.fileSet, abs) {
		return fmt.Errorf("zerolog: duplicate file output %q", abs)
	}
	c.fileSet = append(c.fileSet, abs)
	return nil
}

func defaultLoggerCfg() loggerCfg {
	return loggerCfg{
		level:      zerolog.InfoLevel,
		timestamp:  true,
		timeFormat: time.RFC3339,
	}
}

// Option configures the logger built by New.
type Option func(*loggerCfg) error

// WithConsole adds a console output with pretty (human-readable) formatting enabled by default.
func WithConsole(w io.Writer, opts ...OutputOption) Option {
	return func(c *loggerCfg) error {
		if err := c.trackWriter(w); err != nil {
			return err
		}
		cfg := defaultOutputCfg()
		cfg.pretty = true
		for _, o := range opts {
			o(&cfg)
		}
		c.outputs = append(c.outputs, buildLevelWriter(w, cfg))
		return nil
	}
}

// WithFile adds a file output at path. The file is created if absent, appended to otherwise.
// JSON formatting is used by default. File permissions are 0600.
func WithFile(path string, opts ...OutputOption) Option {
	return func(c *loggerCfg) error {
		if err := c.trackFile(path); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		c.closers = append(c.closers, f)
		cfg := defaultOutputCfg()
		for _, o := range opts {
			o(&cfg)
		}
		c.outputs = append(c.outputs, buildLevelWriter(f, cfg))
		return nil
	}
}

// WithWriter adds any io.Writer as an output with JSON formatting.
func WithWriter(w io.Writer, opts ...OutputOption) Option {
	return func(c *loggerCfg) error {
		if err := c.trackWriter(w); err != nil {
			return err
		}
		cfg := defaultOutputCfg()
		for _, o := range opts {
			o(&cfg)
		}
		c.outputs = append(c.outputs, buildLevelWriter(w, cfg))
		return nil
	}
}

// WithLevel sets the global minimum log level (default: Info).
// Per-output levels set via OutputLevel can only restrict further, not relax this limit.
func WithLevel(level zerolog.Level) Option {
	return func(c *loggerCfg) error {
		c.level = level
		return nil
	}
}

// WithCaller appends file:line caller info to every log entry.
func WithCaller() Option {
	return func(c *loggerCfg) error {
		c.caller = true
		return nil
	}
}

// WithCallerSkip sets extra stack frames to skip when reporting caller location.
// Use this when the logger is wrapped in a helper layer.
func WithCallerSkip(n int) Option {
	return func(c *loggerCfg) error {
		c.callerSkip = n
		return nil
	}
}

// WithTimestamp enables the automatic timestamp field (enabled by default).
func WithTimestamp() Option {
	return func(c *loggerCfg) error {
		c.timestamp = true
		return nil
	}
}

// WithNoTimestamp disables the automatic timestamp field.
func WithNoTimestamp() Option {
	return func(c *loggerCfg) error {
		c.timestamp = false
		return nil
	}
}

// WithTimeFormat sets the timestamp format (default: time.RFC3339).
// This writes to zerolog.TimeFieldFormat — a package-level global that affects
// the entire process, not just this logger instance.
func WithTimeFormat(format string) Option {
	return func(c *loggerCfg) error {
		c.timeFormat = format
		return nil
	}
}

// WithTimeUTC forces all timestamps to UTC.
// This replaces zerolog.TimestampFunc — a package-level global.
func WithTimeUTC() Option {
	return func(c *loggerCfg) error {
		c.timeUTC = true
		return nil
	}
}

// WithStr adds a static string field to every log entry.
func WithStr(key, value string) Option {
	return func(c *loggerCfg) error {
		c.ctxFields = append(c.ctxFields, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Str(key, value)
		})
		return nil
	}
}

// WithField adds a static field of any type to every log entry.
func WithField(key string, value any) Option {
	return func(c *loggerCfg) error {
		c.ctxFields = append(c.ctxFields, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Interface(key, value)
		})
		return nil
	}
}

// New builds a configured *zerolog.Logger. It also returns a cleanup function that
// closes any log files opened during construction (in reverse order). Call the cleanup
// on shutdown to ensure all buffered writes are flushed.
//
// Example:
//
//	logger, cleanup, err := log.New(
//	    log.WithConsole(os.Stderr, log.OutputLevel(zerolog.DebugLevel)),
//	    log.WithFile("/var/log/app.log", log.OutputLevel(zerolog.WarnLevel)),
//	    log.WithStr("service", "my-app"),
//	)
//	if err != nil { ... }
//	defer cleanup()
func New(opts ...Option) (*zerolog.Logger, func() error, error) {
	cfg := defaultLoggerCfg()
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, nil, err
		}
	}

	var w io.Writer
	switch len(cfg.outputs) {
	case 0:
		w = io.Discard
	case 1:
		w = cfg.outputs[0]
	default:
		ws := make([]io.Writer, len(cfg.outputs))
		for i, lw := range cfg.outputs {
			ws[i] = lw
		}
		w = zerolog.MultiLevelWriter(ws...)
	}

	// zerolog uses package-level globals — concurrent calls to New will race.
	zerolog.TimeFieldFormat = cfg.timeFormat
	zerolog.ErrorMarshalFunc = oopszerolog.OopsMarshalFunc
	zerolog.ErrorStackMarshaler = oopszerolog.OopsStackMarshaller
	if cfg.timeUTC {
		zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	}

	logger := zerolog.New(w).Level(cfg.level)

	ctx := logger.With()
	if cfg.timestamp {
		ctx = ctx.Timestamp()
	}
	switch {
	case cfg.caller && cfg.callerSkip > 0:
		ctx = ctx.CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + cfg.callerSkip)
	case cfg.caller:
		ctx = ctx.Caller()
	}
	for _, fn := range cfg.ctxFields {
		ctx = fn(ctx)
	}

	cleanup := func() error { return nil }
	if len(cfg.closers) > 0 {
		closers := cfg.closers
		cleanup = func() error {
			var errs []error
			for i := len(closers) - 1; i >= 0; i-- {
				if err := closers[i].Close(); err != nil {
					errs = append(errs, err)
				}
			}
			return errors.Join(errs...)
		}
	}

	result := ctx.Logger()
	return &result, cleanup, nil
}
