package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hansmi/baamhackl/internal/testutil"
	"go.uber.org/zap/zapcore"
)

func TestBuildLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")

	logger, err := buildLogger(zapcore.DebugLevel, []string{"file:" + path})
	if err != nil {
		t.Errorf("buildLogger() failed: %v", err)
	}

	logger.Info("test message")

	if st := testutil.MustLstat(t, path); st.Size() < 10 {
		t.Errorf("Log file too small: %d bytes", st.Size())
	}
}

func TestSeedPseudoRand(t *testing.T) {
	rand.Seed(0)

	seedPseudoRand()
}
