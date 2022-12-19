package watch

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/testutil"
)

func TestMakeHandlerTimeBuckets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		timeout time.Duration
		want    []float64
	}{
		{
			name: "zero",
			want: []float64{1},
		},
		{
			name:    "second",
			timeout: time.Second,
			want:    []float64{1},
		},
		{
			name:    "minute",
			timeout: time.Minute,
			want:    []float64{1, 2, 4, 6, 10, 15, 24, 38, 60},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.HandlerDefaults
			cfg.Path = t.TempDir()
			cfg.Timeout = tc.timeout

			got := makeHandlerTimeBuckets(&cfg)

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateApprox(0, 0.1)); diff != "" {
				t.Errorf("Bucket diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReportProcessState(t *testing.T) {
	cfg := config.HandlerDefaults
	cfg.Path = t.TempDir()

	mc := newHandlerMetricsCollector(newHandler(&cfg))

	testutil.CollectAndCompare(t, mc, `
		# HELP handler_info Information about the handler.
		# TYPE handler_info gauge
		handler_info{path="`+cfg.Path+`"} 1
		# HELP command_exit_code_total Number of times each command exit code occurred.
		# TYPE command_exit_code_total counter
		command_exit_code_total{code="0"} 0
		# HELP command_user_time Histogram with the user space time taken by commands.
		# TYPE command_user_time histogram
		command_user_time_bucket{le="1"} 0
		command_user_time_bucket{le="2"} 0
		command_user_time_bucket{le="6"} 0
		command_user_time_bucket{le="15"} 0
		command_user_time_bucket{le="38"} 0
		command_user_time_bucket{le="95"} 0
		command_user_time_bucket{le="235"} 0
		command_user_time_bucket{le="583"} 0
		command_user_time_bucket{le="1449"} 0
		command_user_time_bucket{le="3600"} 0
		command_user_time_sum 0
		command_user_time_count 0
		`,
		"handler_info",
		"command_exit_code_total",
		"command_user_time",
	)

	mc.ReportProcessState(23, 3*time.Minute, 2*time.Minute, time.Minute)

	testutil.CollectAndCompare(t, mc, `
		# HELP command_exit_code_total Number of times each command exit code occurred.
		# TYPE command_exit_code_total counter
		command_exit_code_total{code="0"} 0
		command_exit_code_total{code="23"} 1
		# HELP command_wall_time Histogram with the time taken by commands from start to end.
		# TYPE command_wall_time histogram
		command_wall_time_bucket{le="1"} 0
		command_wall_time_bucket{le="2"} 0
		command_wall_time_bucket{le="6"} 0
		command_wall_time_bucket{le="15"} 0
		command_wall_time_bucket{le="38"} 0
		command_wall_time_bucket{le="95"} 0
		command_wall_time_bucket{le="235"} 1
		command_wall_time_bucket{le="583"} 1
		command_wall_time_bucket{le="1449"} 1
		command_wall_time_bucket{le="3600"} 1
		command_wall_time_sum 180
		command_wall_time_count 1
		# HELP command_user_time Histogram with the user space time taken by commands.
		# TYPE command_user_time histogram
		command_user_time_bucket{le="1"} 0
		command_user_time_bucket{le="2"} 0
		command_user_time_bucket{le="6"} 0
		command_user_time_bucket{le="15"} 0
		command_user_time_bucket{le="38"} 0
		command_user_time_bucket{le="95"} 0
		command_user_time_bucket{le="235"} 1
		command_user_time_bucket{le="583"} 1
		command_user_time_bucket{le="1449"} 1
		command_user_time_bucket{le="3600"} 1
		command_user_time_sum 120
		command_user_time_count 1
		# HELP command_system_time Histogram with the system time taken by commands.
		# TYPE command_system_time histogram
		command_system_time_bucket{le="1"} 0
		command_system_time_bucket{le="2"} 0
		command_system_time_bucket{le="6"} 0
		command_system_time_bucket{le="15"} 0
		command_system_time_bucket{le="38"} 0
		command_system_time_bucket{le="95"} 1
		command_system_time_bucket{le="235"} 1
		command_system_time_bucket{le="583"} 1
		command_system_time_bucket{le="1449"} 1
		command_system_time_bucket{le="3600"} 1
		command_system_time_bucket{le="+Inf"} 1
		command_system_time_sum 60
		command_system_time_count 1
		`,
		"command_exit_code_total",
		"command_wall_time",
		"command_user_time",
		"command_system_time",
	)
}
