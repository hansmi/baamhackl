package sendfilechanges

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/watchman"
)

func TestReadInput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  []watchman.FileChange
	}{
		{name: "empty", input: `[]`},
		{
			name:  "empty entry",
			input: `[{}]`,
			want:  []watchman.FileChange{{}},
		},
		{
			name: "two entries with unknown field",
			input: `[{
				"name": "test.txt",
				"size": 1234,
				"_unknown_something": false,
				"mtime_us": 1015240953000000,
				"cclock": "abcdef"
			}, {
				"name": "another.txt",
				"size": 987,
				"mtime_us": 1577882096000000,
				"cclock": "xyz",
				"_unknown_field": []
			}]`,
			want: []watchman.FileChange{
				{
					Name:   "test.txt",
					Size:   1234,
					MTime:  time.Date(2002, time.March, 4, 11, 22, 33, 0, time.UTC),
					CClock: "abcdef",
				},
				{
					Name:   "another.txt",
					Size:   987,
					MTime:  time.Date(2020, time.January, 1, 12, 34, 56, 0, time.UTC),
					CClock: "xyz",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := readInput(strings.NewReader(tc.input)); err != nil {
				t.Errorf("readInput() failed: %v", err)
			} else if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("readInput() diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadInputError(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  error
	}{
		{
			name: "empty",
			want: io.EOF,
		},
		{
			name:  "malformed with EOF",
			input: `[{`,
			want:  io.ErrUnexpectedEOF,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readInput(strings.NewReader(tc.input))

			if diff := cmp.Diff(tc.want, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadInputRoundtrip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input []watchman.FileChange
	}{
		{name: "empty"},
		{
			name: "one entry",
			input: []watchman.FileChange{
				{
					Name:   "test.txt",
					Size:   1234,
					MTime:  time.Date(2002, time.March, 4, 11, 22, 33, 0, time.UTC),
					CClock: "0",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if data, err := json.Marshal(tc.input); err != nil {
				t.Errorf("Marshal(%v) failed: %v", tc.input, err)
			} else if got, err := readInput(bytes.NewReader(data)); err != nil {
				t.Errorf("readInput() failed: %v", err)
			} else if diff := cmp.Diff(tc.input, got); diff != "" {
				t.Errorf("readInput() diff (-want +got):\n%s", diff)
			}
		})
	}
}
