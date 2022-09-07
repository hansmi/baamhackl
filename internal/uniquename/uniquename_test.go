package uniquename

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jonboulle/clockwork"
	"golang.org/x/sys/unix"
)

func setupFakeRuntime(t *testing.T, clock clockwork.Clock) {
	t.Helper()

	var randValue int32 = 0xf100

	t.Cleanup(SetRuntime(Runtime{
		Clock: clock,
		Loc:   time.UTC,
		RandInt: func() int32 {
			value := randValue
			randValue++
			return value
		},
	}))
}

func TestGenerator(t *testing.T) {
	timePrefixDisabledOpts := DefaultOptions
	timePrefixDisabledOpts.TimePrefixEnabled = false

	for _, tc := range []struct {
		name    string
		input   string
		opts    Options
		want    []string
		wantErr error
	}{
		{
			name:    "empty",
			opts:    DefaultOptions,
			wantErr: os.ErrInvalid,
		},
		{
			name:    "input ends in slash",
			input:   "path/to/",
			opts:    DefaultOptions,
			wantErr: os.ErrInvalid,
		},
		{
			name:  "name only",
			input: "testname",
			opts:  DefaultOptions,
			want: []string{
				"2018-10-01T020304 testname",
				"2018-10-01T020304 testname (f100)",
				"2018-10-01T020304 testname (f101)",
			},
		},
		{
			name:  "name only without timestamp",
			input: "testname",
			opts:  timePrefixDisabledOpts,
			want: []string{
				"testname",
				"testname (20181001020304)",
				"testname (f100)",
				"testname (f101)",
			},
		},
		{
			name:  "ext only",
			input: ".hidden",
			opts:  DefaultOptions,
			want: []string{
				"2018-10-01T020304 .hidden",
				"2018-10-01T020304 .hidden (f100)",
			},
		},
		{
			name:  "ext only without timestamp",
			input: ".hidden",
			opts:  timePrefixDisabledOpts,
			want: []string{
				".hidden",
				".hidden (20181001020304)",
				".hidden (f100)",
			},
		},
		{
			name:  "dir without timestamp",
			input: "path/to/file.txt",
			opts:  timePrefixDisabledOpts,
			want: []string{
				"path/to/file.txt",
				"path/to/file (20181001020304).txt",
				"path/to/file (f100).txt",
				"path/to/file (f101).txt",
			},
		},
		{
			name:  "dir with only ext without timestamp",
			input: "path/to/.config",
			opts:  timePrefixDisabledOpts,
			want: []string{
				"path/to/.config",
				"path/to/.config (20181001020304)",
				"path/to/.config (f100)",
				"path/to/.config (f101)",
			},
		},
		{
			name:  "trimmed name",
			input: strings.Repeat("a", unix.NAME_MAX),
			opts:  timePrefixDisabledOpts,
			want: []string{
				strings.Repeat("a", unix.NAME_MAX),
				strings.Repeat("a", unix.NAME_MAX-len(" (20181001020304)")) + " (20181001020304)",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fc := clockwork.NewFakeClockAt(time.Date(2018, time.October, 1, 2, 3, 4, 0, time.UTC))
			setupFakeRuntime(t, fc)

			g, err := New(tc.input, tc.opts)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				var got []string

				for path, ok := g.Next(); ok && len(got) < len(tc.want); path, ok = g.Next() {
					got = append(got, path)
				}

				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("Generated name diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestExtractTime(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		opts    Options
		want    time.Time
		wantErr error
	}{
		{
			name:    "empty",
			input:   "",
			opts:    DefaultOptions,
			wantErr: ErrMissingTime,
		},
		{
			name:    "no timestamp",
			input:   "hello world",
			opts:    DefaultOptions,
			wantErr: ErrMissingTime,
		},
		{
			name:    "space only",
			input:   "\n \t",
			opts:    DefaultOptions,
			wantErr: ErrMissingTime,
		},
		{
			name:  "full inclusive timezone",
			input: "2002-03-04T112233+0000",
			opts:  DefaultOptions,
			want:  time.Date(2002, time.March, 4, 11, 22, 33, 0, time.UTC),
		},
		{
			name:  "short",
			input: "2001-08-30T112233",
			opts:  DefaultOptions,
			want:  time.Date(2001, time.August, 30, 11, 22, 33, 0, time.Local),
		},
		{
			name:  "explicit timezone",
			input: "1999-03-21T112233-0800",
			opts:  DefaultOptions,
			want:  time.Date(1999, time.March, 21, 11, 22, 33, 0, time.FixedZone("UTC-8", -8*60*60)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractTime(tc.input, tc.opts)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("ExtractTime(%q) diff (-want +got):\n%s", tc.input, diff)
				}
			}
		})
	}
}
