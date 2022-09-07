package cmdutil

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/subcommands"
)

func TestExecuteStatus(t *testing.T) {
	for _, tc := range []struct {
		name       string
		err        error
		want       subcommands.ExitStatus
		wantOutput string
	}{
		{name: "success"},
		{
			name:       "error",
			err:        errors.New("fake error"),
			want:       subcommands.ExitFailure,
			wantOutput: "Error: fake error\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			got := executeStatus(&buf, tc.err)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("executeStatus() diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantOutput, buf.String()); diff != "" {
				t.Errorf("Output diff (-want +got):\n%s", diff)
			}
		})
	}
}
