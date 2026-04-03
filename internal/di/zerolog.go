package di

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/samber/do/v2"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	applog "github.com/medincident/medincident-zitadel-actions/internal/log"
)

// ProvideZerolog is a samber/do provider for *zerolog.Logger.
func ProvideZerolog(injector do.Injector) (*zerolog.Logger, error) {
	logConfig, err := do.Invoke[*config.ZerologConfig](injector)
	if err != nil {
		return nil, err
	}
	logger, _, err := buildZerolog(logConfig)
	// TODO: wire the cleanup func to application shutdown so log files are flushed and closed.
	return logger, err
}

func buildZerolog(cfg *config.ZerologConfig) (*zerolog.Logger, func() error, error) {
	eb := oops.In("do/zerolog")

	var opts []applog.Option

	if cfg.Level != "" {
		level, err := zerolog.ParseLevel(string(cfg.Level))
		if err != nil {
			return nil, nil, eb.Code("invalid_level").Errorf("invalid log level %q", cfg.Level)
		}
		opts = append(opts, applog.WithLevel(level))
	}

	if cfg.Caller {
		opts = append(opts, applog.WithCaller())
	}
	if cfg.CallerSkip > 0 {
		opts = append(opts, applog.WithCallerSkip(cfg.CallerSkip))
	}

	if cfg.Timestamp != nil && !*cfg.Timestamp {
		opts = append(opts, applog.WithNoTimestamp())
	}

	if cfg.TimeFormat != "" {
		opts = append(opts, applog.WithTimeFormat(cfg.TimeFormat))
	}

	if cfg.TimeUTC {
		opts = append(opts, applog.WithTimeUTC())
	}

	for key, value := range cfg.Fields {
		opts = append(opts, applog.WithStr(key, value))
	}

	for idx := range cfg.Outputs {
		out := &cfg.Outputs[idx]
		outputOpts, err := buildOutputOptions(out)
		if err != nil {
			return nil, nil, eb.With("output_index", idx).Wrap(err)
		}

		switch out.Type {
		case config.ZerologOutputTypeConsole:
			w := os.Stderr
			if out.Target == config.ZerologConsoleTargetStdout {
				w = os.Stdout
			}
			opts = append(opts, applog.WithConsole(w, outputOpts...))

		case config.ZerologOutputTypeFile:
			if out.Path == "" {
				return nil, nil, eb.Code("missing_file_path").With("output_index", idx).
					Errorf("file output requires a non-empty path")
			}
			opts = append(opts, applog.WithFile(out.Path, outputOpts...))

		default:
			return nil, nil, eb.Code("unknown_output_type").With("output_index", idx).
				Errorf("unknown output type %q", out.Type)
		}
	}

	return applog.New(opts...)
}

func buildOutputOptions(out *config.ZerologOutputConfig) ([]applog.OutputOption, error) {
	eb := oops.In("do/zerolog")

	var opts []applog.OutputOption

	if out.Level != "" {
		level, err := zerolog.ParseLevel(string(out.Level))
		if err != nil {
			return nil, eb.Code("invalid_output_level").Errorf("invalid output level %q", out.Level)
		}
		opts = append(opts, applog.OutputLevel(level))
	}

	if out.Pretty != nil {
		if *out.Pretty {
			opts = append(opts, applog.OutputPretty())
		} else {
			opts = append(opts, applog.OutputJSON())
		}
	}

	if out.NoColor {
		opts = append(opts, applog.OutputNoColor())
	}

	if out.TimeFormat != "" {
		opts = append(opts, applog.OutputTimeFormat(out.TimeFormat))
	}

	if len(out.PartsOrder) > 0 {
		opts = append(opts, applog.OutputPartsOrder(out.PartsOrder...))
	}

	if len(out.PartsExclude) > 0 {
		opts = append(opts, applog.OutputPartsExclude(out.PartsExclude...))
	}

	return opts, nil
}
