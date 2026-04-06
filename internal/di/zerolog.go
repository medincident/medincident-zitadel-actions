package di

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"github.com/samber/oops"
	oopszerolog "github.com/samber/oops/loggers/zerolog"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

// loggerWrapper holds the logger and a cleanup function that closes file handles.
// It implements do.ShutdownerWithContextAndError so samber/do calls Shutdown
// when the injector shuts down.
type loggerWrapper struct {
	logger  *zerolog.Logger
	cleanup func() error
}

func (w *loggerWrapper) Shutdown(_ context.Context) error {
	return oops.In("di/zerolog").Code("cleanup_failed").Wrap(w.cleanup())
}

// ProvideLoggerWrapper is a samber/do provider for *loggerWrapper.
func ProvideLoggerWrapper(injector do.Injector) (*loggerWrapper, error) {
	cfg := do.MustInvoke[*config.Config](injector)
	logger, cleanup, err := buildZerolog(&cfg.Zerolog)
	if err != nil {
		return nil, err
	}
	return &loggerWrapper{logger: logger, cleanup: cleanup}, nil
}

// ProvideZerolog is a samber/do provider for *zerolog.Logger.
func ProvideZerolog(injector do.Injector) (*zerolog.Logger, error) {
	w, err := do.Invoke[*loggerWrapper](injector)
	if err != nil {
		return nil, err
	}
	return w.logger, nil
}

func buildZerolog(cfg *config.ZerologConfig) (*zerolog.Logger, func() error, error) {
	eb := oops.In("di/zerolog")

	// Parse global level.
	globalLevel := zerolog.InfoLevel
	if cfg.Level != "" {
		level, err := zerolog.ParseLevel(string(cfg.Level))
		if err != nil {
			return nil, nil, eb.Code("invalid_level").Errorf("invalid log level %q", cfg.Level)
		}
		globalLevel = level
	}

	// Resolve time format.
	timeFormat := time.RFC3339
	if cfg.TimeFormat != "" {
		timeFormat = cfg.TimeFormat
	}

	// Build outputs.
	var (
		outputs   []zerolog.LevelWriter
		closers   []io.Closer
		writers   []*os.File
		filePaths []string
	)

	for idx := range cfg.Outputs {
		out := &cfg.Outputs[idx]

		switch out.Type {
		case config.ZerologOutputTypeConsole:
			w := consoleTarget(out.Target)
			if slices.Contains(writers, w) {
				return nil, nil, eb.Code("duplicate_writer").With("output_index", idx).
					Errorf("duplicate console output for %s", out.Target)
			}
			writers = append(writers, w)
			outputs = append(outputs, buildOutputWriter(w, out, timeFormat))

		case config.ZerologOutputTypeFile:
			if out.Path == "" {
				return nil, nil, eb.Code("missing_file_path").With("output_index", idx).
					Errorf("file output requires a non-empty path")
			}
			abs, err := filepath.Abs(out.Path)
			if err != nil {
				return nil, nil, eb.With("output_index", idx).Wrap(err)
			}
			if slices.Contains(filePaths, abs) {
				return nil, nil, eb.Code("duplicate_file").With("output_index", idx).
					Errorf("duplicate file output %q", abs)
			}
			filePaths = append(filePaths, abs)

			f, err := os.OpenFile(out.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				return nil, nil, eb.With("output_index", idx).Wrap(err)
			}
			closers = append(closers, f)
			outputs = append(outputs, buildOutputWriter(f, out, timeFormat))

		default:
			return nil, nil, eb.Code("unknown_output_type").With("output_index", idx).
				Errorf("unknown output type %q", out.Type)
		}
	}

	// Compose multi-writer.
	var w io.Writer
	switch len(outputs) {
	case 0:
		w = io.Discard
	case 1:
		w = outputs[0]
	default:
		ws := make([]io.Writer, len(outputs))
		for i, lw := range outputs {
			ws[i] = lw
		}
		w = zerolog.MultiLevelWriter(ws...)
	}

	// Set zerolog package-level globals.
	zerolog.TimeFieldFormat = timeFormat
	zerolog.ErrorMarshalFunc = oopszerolog.OopsMarshalFunc
	zerolog.ErrorStackMarshaler = oopszerolog.OopsStackMarshaller
	if cfg.TimeUTC {
		zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	}

	// Build logger.
	logger := zerolog.New(w).Level(globalLevel)
	ctx := logger.With()

	timestamp := true
	if cfg.Timestamp != nil {
		timestamp = *cfg.Timestamp
	}
	if timestamp {
		ctx = ctx.Timestamp()
	}

	switch {
	case cfg.Caller && cfg.CallerSkip > 0:
		ctx = ctx.CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + cfg.CallerSkip)
	case cfg.Caller:
		ctx = ctx.Caller()
	}

	for key, value := range cfg.Fields {
		ctx = ctx.Str(key, value)
	}

	// Build cleanup.
	cleanup := func() error { return nil }
	if len(closers) > 0 {
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

func consoleTarget(target config.ZerologConsoleTarget) *os.File {
	if target == config.ZerologConsoleTargetStdout {
		return os.Stdout
	}
	return os.Stderr
}

func buildOutputWriter(w io.Writer, out *config.ZerologOutputConfig, globalTimeFormat string) zerolog.LevelWriter {
	// Determine if pretty mode.
	pretty := false
	if out.Pretty != nil {
		pretty = *out.Pretty
	} else if out.Type == config.ZerologOutputTypeConsole {
		pretty = true
	}

	// Resolve per-output time format.
	tf := globalTimeFormat
	if out.TimeFormat != "" {
		tf = out.TimeFormat
	}

	var base zerolog.LevelWriter
	if pretty {
		cw := zerolog.ConsoleWriter{
			Out:        w,
			NoColor:    out.NoColor,
			TimeFormat: tf,
		}
		if len(out.PartsOrder) > 0 {
			cw.PartsOrder = out.PartsOrder
		}
		if len(out.PartsExclude) > 0 {
			cw.PartsExclude = out.PartsExclude
		}
		base = zerolog.LevelWriterAdapter{Writer: cw}
	} else {
		base = zerolog.LevelWriterAdapter{Writer: w}
	}

	// Per-output level filtering.
	if out.Level != "" {
		level, err := zerolog.ParseLevel(string(out.Level))
		if err == nil && level > zerolog.TraceLevel {
			return &zerolog.FilteredLevelWriter{Writer: base, Level: level}
		}
	}

	return base
}
