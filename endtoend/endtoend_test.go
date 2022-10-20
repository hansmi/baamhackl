// Package endtoend_test implements a number of tests checking the whole
// program, not just individual parts as unittests do.
//
// The path to the main program is required as an argument, e.g.:
//
//   go test -tags endtoend -- ../baamhackl
//

//go:build endtoend

package endtoend_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hansmi/baamhackl/internal/testutil"
)

func prepareCommand(ctx context.Context, t *testing.T, args []string) *exec.Cmd {
	t.Helper()

	if flag.NArg() < 1 {
		t.Fatalf("Missing command")
	}

	args = append(append([]string(nil), flag.Args()...), args...)

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Env = append([]string(nil), os.Environ()...)
	cmd.Env = append(cmd.Env, "BAAMHACKL_CONFIG_FILE=")

	return cmd
}

func captureOutput(ctx context.Context, t *testing.T, args []string, stdin string) string {
	t.Helper()

	var buf bytes.Buffer

	cmd := prepareCommand(ctx, t, args)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		t.Errorf("Command %q failed: %s\n%s", cmd.Args, err, buf.String())
	} else if buf.Len() > 0 {
		t.Logf("Command %q output:\n%s", cmd.Args, buf.String())
	} else {
		t.Logf("Command %q succeeded", cmd.Args)
	}

	return buf.String()
}

func TestHelp(t *testing.T) {
	var wantRe = regexp.MustCompile(`(?m)^\s*send-file-changes\s`)

	output := captureOutput(context.Background(), t, []string{"help"}, "")

	if !wantRe.MatchString(output) {
		t.Errorf("Command output doesn't match %q:\n%s", wantRe.String(), output)
	}
}

func TestMoveInto(t *testing.T) {
	var wantRe = regexp.MustCompile(`(?im)\sFile moved successfully\s`)

	tmpdir := t.TempDir()
	destDir := testutil.MustMkdir(t, filepath.Join(tmpdir, "dest"))
	srcDir := testutil.MustMkdir(t, filepath.Join(tmpdir, "src"))

	totalCount := 0

	for fileCount := 1; fileCount < 5; fileCount++ {
		args := []string{"move-into", destDir}

		for i := 0; i < fileCount; i++ {
			path := filepath.Join(srcDir, fmt.Sprintf("test%d", i))
			args = append(args, testutil.MustWriteFile(t, path, ""))
		}

		output := captureOutput(context.Background(), t, args, "")

		if !wantRe.MatchString(output) {
			t.Errorf("Command output doesn't match %q:\n%s", wantRe.String(), output)
		}

		totalCount += fileCount
	}

	if entries, err := os.ReadDir(destDir); err != nil {
		t.Error(err)
	} else if len(entries) != totalCount {
		t.Errorf("found %d files, want %d", len(entries), totalCount)
	}
}

func TestSendFileChanges(t *testing.T) {
	args := []string{
		"send-file-changes",
		"-name", "handler-name",
		"-root", t.TempDir(),
		filepath.Join(t.TempDir(), "socket"),
	}

	output := captureOutput(context.Background(), t, args, "[]")

	if strings.TrimSpace(output) != "" {
		t.Errorf("Output is not empty: %q", output)
	}
}

func TestSelftest(t *testing.T) {
	timeout := time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{"selftest", "-timeout", timeout.String()}

	var buf bytes.Buffer

	cmd := prepareCommand(ctx, t, args)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		t.Errorf("Self-test failed: %v\nCommand output:\n%s", err, buf.String())
	}
}
