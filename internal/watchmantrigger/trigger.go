package watchmantrigger

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"time"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/relpath"
)

// Build the Watchman query expression for a handler.
func makeQueryExpression(h config.Handler, ignoreDirs []string) []any {
	expr := []any{
		"allof",
		[]string{"exists"},
		[]string{"type", "f"},
	}

	if !h.Recursive {
		expr = append(expr, []any{"dirname", "", []any{"depth", "eq", 0}})
	}

	for _, i := range ignoreDirs {
		expr = append(expr, []any{"not", []string{"dirname", i}})
	}

	if !h.IncludeHidden {
		expr = append(expr, []any{"not", []string{"match", ".*", "basename"}})
	}

	if h.MinSizeBytes > 0 {
		expr = append(expr, []any{"size", "ge", h.MinSizeBytes})
	}

	if h.MaxSizeBytes > 0 {
		expr = append(expr, []any{"size", "le", h.MaxSizeBytes})
	}

	return expr
}

type triggerConfig struct {
	configFilePath string
	configData     map[string]any
	expression     any
}

func newTriggerConfig(h config.Handler) (*triggerConfig, error) {
	var ignoreDirs []string

	for _, i := range []string{
		h.JournalDir,
		h.SuccessDir,
		h.FailureDir,
	} {
		if r, err := relpath.Resolve(h.Path, i); err != nil {
			return nil, err
		} else if r.Contained() {
			ignoreDirs = append(ignoreDirs, r.Relative)
		}
	}

	sort.Strings(ignoreDirs)

	return &triggerConfig{
		configFilePath: filepath.Join(h.Path, configFileLocalScope),

		// https://facebook.github.io/watchman/docs/config.html
		configData: map[string]any{
			// Number of milliseconds the filesystem should be idle before
			// dispatching triggers.
			"settle": int(h.SettleDuration.Milliseconds()),

			// Matching directories are completely ignored.
			"ignore_dirs": ignoreDirs,

			// Deleted files and directories older than this are periodically
			// pruned from the internal view of the filesystem.
			"gc_age_seconds": int(time.Hour.Seconds()),

			// How often to check for and prune out deleted nodes per the
			// gc_age_seconds option.
			"gc_interval_seconds": int(time.Hour.Seconds()),

			// Number of seconds a watch can remain idle before becoming
			// a candidate for reaping.
			"idle_reap_age_seconds": int(time.Minute.Seconds()),

			// Avoid producing recrawl-related warnings.
			"suppress_recrawl_warnings": true,
		},

		expression: makeQueryExpression(h, ignoreDirs),
	}, nil
}

func (c *triggerConfig) configDataJSON() ([]byte, error) {
	return json.MarshalIndent(c.configData, "", "  ")
}
