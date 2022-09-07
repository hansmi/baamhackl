package cmdemu

import (
	"bytes"
	"errors"
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		name       string
		w          *Wrapper
		want       int
		wantOutput string
	}{
		{
			name:       "empty",
			w:          New(flag.NewFlagSet("", flag.PanicOnError)),
			want:       ExitUsage,
			wantOutput: "Unknown command \"empty\"\n",
		},
		{
			name: "success",
			w: func() *Wrapper {
				w := New(flag.NewFlagSet("", flag.PanicOnError))
				w.Register(Command{
					Name: "success",
					Execute: func([]string) error {
						return nil
					},
				})
				return w
			}(),
			want: ExitSuccess,
		},
		{
			name: "failure",
			w: func() *Wrapper {
				w := New(flag.NewFlagSet("", flag.PanicOnError))
				w.Register(Command{
					Name: "failure",
					Execute: func([]string) error {
						return errors.New("test error")
					},
				})
				return w
			}(),
			want:       ExitUnavailable,
			wantOutput: "Error: test error\n",
		},
		{
			name: "customcode",
			w: func() *Wrapper {
				w := New(flag.NewFlagSet("", flag.PanicOnError))
				w.Register(Command{
					Name: "customcode",
					Execute: func([]string) error {
						return ExitCodeError(123)
					},
				})
				return w
			}(),
			want: 123,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			got := tc.w.run(&buf, tc.name, nil)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("run() result diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantOutput, buf.String()); diff != "" {
				t.Errorf("Output diff (-want +got):\n%s", diff)
			}
		})
	}
}
