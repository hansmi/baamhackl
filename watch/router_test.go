package watch

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/service"
	"github.com/hansmi/baamhackl/internal/watchman"
)

func TestRouterStartStop(t *testing.T) {
	r := newRouter(routerOptions{})
	r.start(10)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r.stop(ctx); err != nil {
		t.Errorf("stop() failed: %v", err)
	}
}

func TestRouterFileChangedFailure(t *testing.T) {
	tmpdir := t.TempDir()

	for _, tc := range []struct {
		req     service.FileChangedRequest
		wantErr *regexp.Regexp
	}{
		{
			wantErr: regexp.MustCompile(`^missing handler name\b`),
		},
		{
			req: service.FileChangedRequest{
				HandlerName: "something",
				RootDir:     tmpdir,
			},
			wantErr: regexp.MustCompile(`^missing\b.*changed file\b`),
		},
		{
			req: service.FileChangedRequest{
				HandlerName: "missing",
				RootDir:     tmpdir,
				Change:      watchman.FileChange{Name: "foo"},
			},
			wantErr: regexp.MustCompile(`^handler\b.*not found\b`),
		},
		{
			req: service.FileChangedRequest{
				HandlerName: "handler",
				RootDir:     ".",
				Change:      watchman.FileChange{Name: "name"},
			},
			wantErr: regexp.MustCompile(`^root\b.*\babsolute path\b`),
		},
		{
			req: service.FileChangedRequest{
				HandlerName: "handler",
				RootDir:     tmpdir,
				Change:      watchman.FileChange{Name: "/path/to/file"},
			},
			wantErr: regexp.MustCompile(`^filename\b.*\brelative path\b`),
		},
		{
			req: service.FileChangedRequest{
				HandlerName: "configured",
				RootDir:     t.TempDir(),
				Change:      watchman.FileChange{Name: "aaa"},
			},
			wantErr: regexp.MustCompile(`^root directory\b.*\bdiffers\b`),
		},
	} {
		r := newRouter(routerOptions{
			handlers: []*config.Handler{
				{
					Name: "configured",
					Path: tmpdir,
				},
			},
		})
		err := r.FileChanged(tc.req)

		if err == nil || !tc.wantErr.MatchString(err.Error()) {
			t.Errorf("FileChanged(%#v) returned %q, want match for %q", tc.req, err, tc.wantErr.String())
		}
	}
}

func TestRouterMultipleChanges(t *testing.T) {
	tmpdir := t.TempDir()

	r := newRouter(routerOptions{
		handlers: []*config.Handler{
			{
				Name: "compressor",
				Path: tmpdir,
			},
		},
	})

	req := service.FileChangedRequest{
		HandlerName: "compressor",
		RootDir:     tmpdir,
	}

	fileVariations := []string{
		"path/to/file",
		"./path/to/file",
		"path///to/./file",
	}

	h := r.handlerByName["compressor"]

	for i := 0; i < 10; i++ {
		req.Change.Name = fileVariations[i%len(fileVariations)]

		if err := r.FileChanged(req); err != nil {
			t.Errorf("FileChanged(%+v) failed: %v", req, err)
		}

		h.mu.Lock()
		if diff := cmp.Diff(1, len(h.pending)); diff != "" {
			t.Errorf("Pending changes count (-want +got):\n%s", diff)
		}
		h.mu.Unlock()
	}
}
