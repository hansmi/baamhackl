package config

import (
	"time"

	"github.com/goccy/go-yaml"
)

// Default configuration for a handler.
var HandlerDefaults = Handler{
	Timeout:           time.Hour,
	SettleDuration:    time.Second,
	RetryCount:        2,
	RetryDelayInitial: 15 * time.Minute,
	RetryDelayFactor:  1.5,
	RetryDelayMax:     time.Hour,
	JournalDir:        "_/journal",
	JournalRetention:  24 * 7 * time.Hour,
	SuccessDir:        "_/success",
	FailureDir:        "_/failure",
}

type Handler struct {
	// Name of the trigger registered in Watchman.
	Name string `yaml:"name" validate:"required"`

	// Path to root directory.
	Path string `yaml:"path" validate:"required"`

	// Command executed when file changes are detected. Arguments are visible
	// in log files and shouldn't contain confidential information such as
	// passwords or access tokens.
	Command []string `yaml:"command" validate:"required,gte=1"`

	// Timeout for executing command.
	Timeout time.Duration `yaml:"timeout" validate:"min=0"`

	// Observe input directory recursively (excluding the infrastructure
	// directories).
	Recursive bool `yaml:"recursive"`

	// Whether to process files with names starting with a dot (".").
	IncludeHidden bool `yaml:"include_hidden"`

	// Minimum file size for running command
	MinSizeBytes uint64 `yaml:"min_size_bytes"`

	// Maximum file size for running command
	MaxSizeBytes uint64 `yaml:"max_size_bytes"`

	// Amount of time the filesystem should be idle before dispatching
	// triggers.
	SettleDuration time.Duration `yaml:"settle_duration" validate:"min=0"`

	// Number of times a failing command should be retried. Set to 0 to make
	// the first failure permanent.
	RetryCount int `yaml:"retry_count" validate:"min=0"`

	// Amount of time to wait between retry attempts. A small and random amount
	// of fuzzing is always applied.
	RetryDelayInitial time.Duration `yaml:"retry_delay_initial" validate:"required,gt=0"`

	// Backoff factor to apply between attempts after the first retry. Use 1 to
	// always use the same delay.
	RetryDelayFactor float64 `yaml:"retry_delay_factor" validate:"required,min=1"`

	// Maximum amount of time waiting between retry attempts. Use 0s for no
	// limit.
	RetryDelayMax time.Duration `yaml:"retry_delay_max" validate:"eq=0|gtefield=RetryDelayInitial"`

	// Directory into which journal entries are written.
	JournalDir string `yaml:"journal_dir" validate:"required"`

	// How long to keep journal entries.
	JournalRetention time.Duration `yaml:"journal_retention" validate:"min=1h|gtefield=Timeout|gtefield=RetryDelayMax"`

	// Directory into which files are moved whose processing succeeded.
	SuccessDir string `yaml:"success_dir" validate:"required"`

	// Directory into which files are moved whose processing failed.
	FailureDir string `yaml:"failure_dir" validate:"required"`
}

var _ yaml.InterfaceUnmarshaler = (*Handler)(nil)

func (h *Handler) UnmarshalYAML(unmarshal func(any) error) error {
	*h = HandlerDefaults

	type handler Handler

	return unmarshal((*handler)(h))
}
