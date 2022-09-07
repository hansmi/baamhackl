package config

import (
	"regexp"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	marshalTestSlice{
		{
			name: "minimal",
			input: `
---
name: test
path: foo/bar
command: ["/bin/false"]
`,
			want: Handler{
				Name:              "test",
				Path:              "foo/bar",
				Command:           []string{"/bin/false"},
				Timeout:           time.Hour,
				SettleDuration:    time.Second,
				RetryCount:        2,
				RetryDelayInitial: 15 * time.Minute,
				RetryDelayFactor:  1.5,
				RetryDelayMax:     time.Hour,
				JournalDir:        "_/journal",
				JournalRetention:  7 * 24 * time.Hour,
				SuccessDir:        "_/success",
				FailureDir:        "_/failure",
			},
		},
		{
			name: "customized",
			input: `
---
name: custom
path: /abs/path
command: ["/bin/true", "arg"]
timeout: 3m17s
recursive: true
include_hidden: true
settle_duration: 3s
retry_count: 123
retry_delay_initial: 7m3s
retry_delay_factor: 7
retry_delay_max: 2h
journal_dir: /another/dir
journal_retention: 2h7s
success_dir: /another/success
failure_dir: /another/failure
`,
			want: Handler{
				Name:              "custom",
				Path:              "/abs/path",
				Command:           []string{"/bin/true", "arg"},
				Timeout:           3*time.Minute + 17*time.Second,
				Recursive:         true,
				IncludeHidden:     true,
				SettleDuration:    3 * time.Second,
				RetryCount:        123,
				RetryDelayInitial: 7*time.Minute + 3*time.Second,
				RetryDelayFactor:  7,
				RetryDelayMax:     2 * time.Hour,
				JournalDir:        "/another/dir",
				JournalRetention:  2*time.Hour + 7*time.Second,
				SuccessDir:        "/another/success",
				FailureDir:        "/another/failure",
			},
		},
		{
			name: "retry max disabled",
			input: `
---
name: zeromax
path: foo/bar
command: ["/bin/true"]
retry_delay_max: 0s
`,
			want: func() Handler {
				o := HandlerDefaults
				o.Name = "zeromax"
				o.Path = "foo/bar"
				o.Command = []string{"/bin/true"}
				o.RetryDelayMax = 0
				return o
			}(),
		},
		{
			name: "retry max lower than initial",
			input: `
---
name: lowmax
path: foo/bar
command: ["/bin/true"]
retry_delay_max: 1m
`,
			want:    Handler{},
			wantErr: regexp.MustCompile(`(?i)\bretry_delay_max\b.*\bfailed\b.*\bgtefield\b`),
		},
		{
			name: "missing command",
			input: `
---
name: no command
path: foo/bar
command: []
`,
			want:    Handler{},
			wantErr: regexp.MustCompile(`(?i)\bcommand\b.*\bfailed\b.*\bgte\b`),
		},
	}.run(t)
}
