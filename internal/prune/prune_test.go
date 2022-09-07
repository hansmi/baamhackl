package prune

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
	"github.com/hansmi/baamhackl/internal/uniquename"
	"github.com/spf13/afero"
)

func TestMakeAgeFilter(t *testing.T) {
	deadline := time.Date(2014, time.April, 10, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name   string
		mtime  time.Time
		accept AcceptFunc
		want   bool
	}{
		{
			name:   "",
			accept: MakeAgeFilter(time.Time{}, uniquename.DefaultOptions),
			want:   true,
		},
		{
			name:   "2014-04-10T010000+0000 newer than deadline",
			mtime:  deadline.Add(time.Hour),
			accept: MakeAgeFilter(deadline, uniquename.DefaultOptions),
		},
		{
			name:   "2014-04-09T000000+0000 older than deadline",
			mtime:  deadline.Add(-time.Hour),
			accept: MakeAgeFilter(deadline, uniquename.DefaultOptions),
			want:   true,
		},
		{
			name:   "no timestamp older than deadline",
			mtime:  deadline.Add(-time.Hour),
			accept: MakeAgeFilter(deadline, uniquename.DefaultOptions),
			want:   true,
		},
		{
			name:   "no timestamp newer than deadline",
			mtime:  deadline.Add(time.Hour),
			accept: MakeAgeFilter(deadline, uniquename.DefaultOptions),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if err := fs.Chtimes(".", tc.mtime, tc.mtime); err != nil {
				t.Error(err)
			}
			fi, err := fs.Stat(".")
			if err != nil {
				t.Error(err)
			}

			got := tc.accept(tc.name, fi)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("accept result diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPrunerAlreadyLocked(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tmpdir := t.TempDir()

	lock := flock.New(filepath.Join(tmpdir, lockName))
	if err := lock.Lock(); err != nil {
		t.Errorf("Lock() failed: %v", err)
	}
	defer lock.Close()

	err := Pruner{
		Dir: tmpdir,
	}.Run(ctx)
	wantErr := ErrUnavailable

	if diff := cmp.Diff(wantErr, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Error diff (-want +got):\n%s", diff)
	}
}

func TestPrunerBadDirectory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	err := Pruner{
		Dir: filepath.Join(t.TempDir(), "missing"),
	}.Run(ctx)
	wantErr := os.ErrNotExist

	if diff := cmp.Diff(wantErr, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Error diff (-want +got):\n%s", diff)
	}
}

func TestPrunerEmptyDirectory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	err := Pruner{
		Dir: t.TempDir(),
	}.Run(ctx)
	var wantErr error

	if diff := cmp.Diff(wantErr, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Error diff (-want +got):\n%s", diff)
	}
}

func TestPrunerFilterCancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tmpdir := t.TempDir()

	for i := 0; i < 10; i++ {
		testutil.MustWriteFile(t, filepath.Join(tmpdir, fmt.Sprint(i)), "")
	}

	err := Pruner{
		Dir: tmpdir,
		Accept: func(string, os.FileInfo) bool {
			cancel()
			return false
		},
	}.Run(ctx)
	wantErr := context.Canceled

	if diff := cmp.Diff(wantErr, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Error diff (-want +got):\n%s", diff)
	}
}

func TestPruner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	fs := afero.NewMemMapFs()

	tmpdir, err := afero.TempDir(fs, "", "")
	if err != nil {
		t.Error(err)
	}

	names := []string{}
	keep := map[string]bool{}

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("dir%d", i)

		if err := fs.Mkdir(filepath.Join(tmpdir, name), os.ModePerm); err != nil {
			t.Error(err)
		}

		names = append(names, name)
		keep[name] = i%2 == 0
	}

	p := Pruner{
		Dir: tmpdir,
		Accept: func(name string, fi os.FileInfo) bool {
			return !keep[name]
		},

		fs: fs,
	}

	if err := p.runLocked(ctx); err != nil {
		t.Errorf("runLocked() failed: %v", err)
	}

	for _, name := range names {
		exists, err := afero.DirExists(fs, filepath.Join(tmpdir, name))
		if err != nil {
			t.Errorf("DirExists() failed: %v", err)
		}

		if diff := cmp.Diff(keep[name], exists); diff != "" {
			t.Errorf("Existence diff for %q (-want +got):\n%s", name, diff)
		}
	}
}

func TestPrunerWithAgeFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	p := Pruner{
		Dir:    ".",
		Accept: MakeAgeFilter(time.Now(), uniquename.DefaultOptions),

		fs: afero.NewMemMapFs(),
	}

	if err := p.runLocked(ctx); err != nil {
		t.Errorf("runLocked() failed: %v", err)
	}
}
