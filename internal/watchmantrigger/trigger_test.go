package watchmantrigger

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/config"
)

func TestNewTriggerConfig(t *testing.T) {
	tmpdir := t.TempDir()

	for _, tc := range []struct {
		name    string
		cfg     config.Handler
		want    *triggerConfig
		wantErr error
	}{
		{
			name: "defaults",
			cfg: func() config.Handler {
				o := config.HandlerDefaults
				o.Path = tmpdir
				return o
			}(),
			want: &triggerConfig{
				configFilePath: filepath.Join(tmpdir, configFileLocalScope),
				configData: map[string]any{
					"gc_age_seconds":        3600,
					"gc_interval_seconds":   3600,
					"idle_reap_age_seconds": 60,
					"ignore_dirs": []string{
						"_/failure",
						"_/journal",
						"_/success",
					},
					"settle":                    1000,
					"suppress_recrawl_warnings": true,
				},
				expression: []any{
					"allof",
					[]string{"exists"},
					[]string{"type", "f"},

					[]any{"dirname", "", []any{"depth", "eq", 0}},

					[]any{"not", []string{"dirname", "_/failure"}},
					[]any{"not", []string{"dirname", "_/journal"}},
					[]any{"not", []string{"dirname", "_/success"}},

					[]any{"not", []string{"match", ".*", "basename"}},
				},
			},
		},
		{
			name: "customized",
			cfg: func() config.Handler {
				o := config.HandlerDefaults
				o.Path = tmpdir
				o.Recursive = !o.Recursive
				o.IncludeHidden = !o.IncludeHidden
				o.MinSizeBytes = 128
				o.MaxSizeBytes = 1024
				return o
			}(),
			want: &triggerConfig{
				configFilePath: filepath.Join(tmpdir, configFileLocalScope),
				configData: map[string]any{
					"gc_age_seconds":        3600,
					"gc_interval_seconds":   3600,
					"idle_reap_age_seconds": 60,
					"ignore_dirs": []string{
						"_/failure",
						"_/journal",
						"_/success",
					},
					"settle":                    1000,
					"suppress_recrawl_warnings": true,
				},
				expression: []any{
					"allof",
					[]string{"exists"},
					[]string{"type", "f"},

					[]any{"not", []string{"dirname", "_/failure"}},
					[]any{"not", []string{"dirname", "_/journal"}},
					[]any{"not", []string{"dirname", "_/success"}},

					[]any{"size", "ge", uint64(128)},
					[]any{"size", "le", uint64(1024)},
				},
			},
		},
		{
			name: "custom dirs",
			cfg: func() config.Handler {
				o := config.HandlerDefaults
				o.Path = tmpdir
				o.JournalDir = "log"
				o.SuccessDir = "foo/../good"
				o.FailureDir = "./bad"
				return o
			}(),
			want: &triggerConfig{
				configFilePath: filepath.Join(tmpdir, configFileLocalScope),
				configData: map[string]any{
					"gc_age_seconds":        3600,
					"gc_interval_seconds":   3600,
					"idle_reap_age_seconds": 60,
					"ignore_dirs": []string{
						"bad",
						"good",
						"log",
					},
					"settle":                    1000,
					"suppress_recrawl_warnings": true,
				},
				expression: []any{
					"allof",
					[]string{"exists"},
					[]string{"type", "f"},

					[]any{"dirname", "", []any{"depth", "eq", 0}},

					[]any{"not", []string{"dirname", "bad"}},
					[]any{"not", []string{"dirname", "good"}},
					[]any{"not", []string{"dirname", "log"}},

					[]any{"not", []string{"match", ".*", "basename"}},
				},
			},
		},
		{
			name: "custom dirs absolute",
			cfg: func() config.Handler {
				o := config.HandlerDefaults
				o.Path = tmpdir
				o.JournalDir = t.TempDir()
				o.SuccessDir = t.TempDir()
				o.FailureDir = t.TempDir()
				return o
			}(),
			want: &triggerConfig{
				configFilePath: filepath.Join(tmpdir, configFileLocalScope),
				configData: map[string]any{
					"gc_age_seconds":            3600,
					"gc_interval_seconds":       3600,
					"idle_reap_age_seconds":     60,
					"ignore_dirs":               []string(nil),
					"settle":                    1000,
					"suppress_recrawl_warnings": true,
				},
				expression: []any{
					"allof",
					[]string{"exists"},
					[]string{"type", "f"},

					[]any{"dirname", "", []any{"depth", "eq", 0}},

					[]any{"not", []string{"match", ".*", "basename"}},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newTriggerConfig(tc.cfg)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("newTriggerConfig() error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty(), cmp.AllowUnexported(triggerConfig{})); diff != "" {
				t.Errorf("newTriggerConfig() result diff (-want +got):\n%s", diff)
			}
		})
	}
}
