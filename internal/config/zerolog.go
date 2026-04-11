package config

// ZerologLevel is a log level string accepted by zerolog (trace|debug|info|warn|error|fatal|panic).
type ZerologLevel string

// ZerologOutputType identifies the kind of output destination.
type ZerologOutputType string

const (
	ZerologOutputTypeConsole ZerologOutputType = "console"
	ZerologOutputTypeFile    ZerologOutputType = "file"
)

// ZerologConsoleTarget selects the standard stream for console output.
type ZerologConsoleTarget string

const (
	ZerologConsoleTargetStdout ZerologConsoleTarget = "stdout"
	ZerologConsoleTargetStderr ZerologConsoleTarget = "stderr"
)

// ZerologOutputConfig configures a single output destination.
type ZerologOutputConfig struct {
	// Type is required: "console" or "file".
	Type ZerologOutputType `yaml:"type"`

	// Level overrides the minimum level for this output only.
	// Empty means inherit the global level.
	Level ZerologLevel `yaml:"level"`

	// Pretty controls human-readable vs JSON formatting.
	Pretty bool `yaml:"pretty"`

	// NoColor disables ANSI color codes (pretty mode only).
	NoColor bool `yaml:"no_color"`

	// TimeFormat overrides the timestamp format for this output.
	// Empty inherits the global TimeFormat.
	TimeFormat string `yaml:"time_format"`

	// PartsOrder sets the field display order in pretty mode.
	// Uses zerolog field name constants (e.g. "time", "level", "message", "caller").
	PartsOrder []string `yaml:"parts_order"`

	// PartsExclude hides specific fields from pretty mode output.
	PartsExclude []string `yaml:"parts_exclude"`

	// Target selects stdout or stderr for console outputs.
	Target ZerologConsoleTarget `yaml:"target"`

	// Path is the log file path for file outputs.
	// The file is created if absent, appended to otherwise.
	Path string `yaml:"path"`
}

// ZerologConfig is the top-level zerolog configuration block.
type ZerologConfig struct {
	// Level is the global minimum log level.
	Level ZerologLevel `yaml:"level"`

	// Caller appends file:line caller info to every log entry.
	Caller bool `yaml:"caller"`

	// CallerSkip adds extra stack frames to skip when Caller is true.
	CallerSkip int `yaml:"caller_skip"`

	// Timestamp enables the automatic timestamp field.
	Timestamp bool `yaml:"timestamp"`

	// TimeFormat is the timestamp format for all outputs.
	// Per-output TimeFormat takes precedence.
	TimeFormat string `yaml:"time_format"`

	// TimeUTC forces all timestamps to UTC.
	TimeUTC bool `yaml:"time_utc"`

	// Fields is a map of static string fields added to every log entry.
	Fields map[string]string `yaml:"fields"`

	// Outputs lists the output destinations. If empty, logs are discarded.
	Outputs []ZerologOutputConfig `yaml:"outputs" validate:"dive"`
}
