package journal

import (
	"context"
	"os"
	"testing"

	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/testutil"
	"go.uber.org/zap/zaptest"
)

func TestJournal(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  config.Handler
	}{
		{
			name: "defaults",
			cfg: func() config.Handler {
				cfg := config.HandlerDefaults
				cfg.Path = t.TempDir()
				return cfg
			}(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			j := New(&tc.cfg)

			testutil.MustNotExist(t, tc.cfg.JournalDir)
			testutil.MustNotExist(t, tc.cfg.SuccessDir)
			testutil.MustNotExist(t, tc.cfg.FailureDir)

			if got, err := j.CreateTaskDir("myhint"); err != nil {
				t.Errorf("CreateTaskDir() failed: %v", err)
			} else if _, err := os.ReadDir(got); err != nil {
				t.Errorf("ReadDir(%q) failed: %v", got, err)
			}

			if err := j.Prune(context.Background(), zaptest.NewLogger(t)); err != nil {
				t.Errorf("Prune() failed: %v", err)
			}
		})
	}
}
