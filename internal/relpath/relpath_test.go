package relpath

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestContained(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input Resolved
		want  bool
	}{
		{name: "empty"},
		{
			name: "good",
			input: Resolved{
				RelativeElems: []string{"foo", "bar"},
			},
			want: true,
		},
		{
			name: "bad",
			input: Resolved{
				RelativeElems: []string{"foo", "..", "bar"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.Contained()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Contained() result diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tmpdir := t.TempDir()

	for _, tc := range []struct {
		name    string
		base    string
		path    string
		want    Resolved
		wantErr error
	}{
		{
			name: "dot",
			base: ".",
			path: ".",
			want: Resolved{
				Path:          ".",
				Base:          ".",
				Relative:      ".",
				RelativeElems: []string{"."},
			},
		},
		{
			name: "absolute and dot",
			base: tmpdir,
			path: ".",
			want: Resolved{
				Path:          tmpdir,
				Base:          tmpdir,
				Relative:      ".",
				RelativeElems: []string{"."},
			},
		},
		{
			name: "relative",
			base: tmpdir,
			path: "foo/bar/baz",
			want: Resolved{
				Path:          filepath.Join(tmpdir, "foo", "bar", "baz"),
				Base:          tmpdir,
				Relative:      filepath.Join("foo", "bar", "baz"),
				RelativeElems: []string{"foo", "bar", "baz"},
			},
		},
		{
			name: "relative with dot",
			base: tmpdir,
			path: "./dir",
			want: Resolved{
				Path:          filepath.Join(tmpdir, "dir"),
				Base:          tmpdir,
				Relative:      "dir",
				RelativeElems: []string{"dir"},
			},
		},
		{
			name: "absolute",
			base: filepath.Join(tmpdir, "base"),
			path: filepath.Join(tmpdir, "another"),
			want: Resolved{
				Path:          filepath.Join(tmpdir, "another"),
				Base:          filepath.Join(tmpdir, "base"),
				Relative:      filepath.Join("..", "another"),
				RelativeElems: []string{"..", "another"},
			},
		},
		{
			name:    "make absolute path relative to relative path",
			base:    ".",
			path:    tmpdir,
			wantErr: cmpopts.AnyError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(tc.base, tc.path)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Resolve(%q, %q) error diff (-want +got):\n%s", tc.base, tc.path, diff)
			}

			if err == nil {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("Resolve(%q, %q) result diff (-want +got):\n%s", tc.base, tc.path, diff)
				}
			}
		})
	}
}
