package watchman

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestFileChangeRoundtrip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input FileChange
	}{
		{name: "empty"},
		{
			name: "one entry",
			input: FileChange{
				Name:   "test.txt",
				Size:   1234,
				MTime:  time.Date(2000, time.February, 3, 11, 22, 33, 0, time.UTC),
				CClock: "0123",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.input)
			if err != nil {
				t.Errorf("Marshal(%v) failed: %v", tc.input, err)
			}

			var got FileChange

			if err := json.Unmarshal(data, &got); err != nil {
				t.Errorf("Unmarshal() failed: %v", err)
			}

			if diff := cmp.Diff(tc.input, got); diff != "" {
				t.Errorf("Unmarshalled diff (-want +got):\n%s", diff)
			}
		})
	}
}
