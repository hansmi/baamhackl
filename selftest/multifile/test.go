package multifile

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/renameio/v2"
	"go.uber.org/zap"
)

func writeInt(path string, value int) error {
	return renameio.WriteFile(path, []byte(fmt.Sprintf("%d\n", value)), 0o600)
}

type generationCounterFile struct {
	path  string
	count int
}

func newGenerationCounterFile(path string) (*generationCounterFile, error) {
	f := &generationCounterFile{
		path: path,
	}

	return f, f.writeCount()
}

func (f *generationCounterFile) writeCount() error {
	return writeInt(f.path, f.count)
}

func (f *generationCounterFile) increment() error {
	f.count++

	return f.writeCount()
}

type file struct {
	name          string
	inputPath     string
	checkmarkPath string
}

func (f *file) finished() (bool, error) {
	if _, err := os.Lstat(f.checkmarkPath); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}

		return false, err
	}

	return true, nil
}

type multifileTest struct {
	dir             string
	statusDir       string
	generation      *generationCounterFile
	generationDelay time.Duration
	counter         int
	pending         map[*file]struct{}
}

func New(dir string) (*multifileTest, error) {
	var err error

	t := &multifileTest{
		counter:         rand.Int(),
		pending:         map[*file]struct{}{},
		generationDelay: time.Second / 4,
	}

	if t.dir, err = os.MkdirTemp(dir, "multifile*"); err != nil {
		return nil, err
	}

	if t.statusDir, err = os.MkdirTemp(dir, "multifilestatus*"); err != nil {
		return nil, err
	}

	if t.generation, err = newGenerationCounterFile(filepath.Join(t.statusDir, "generation")); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *multifileTest) Name() string {
	return "multifile"
}

func (t *multifileTest) HandlerConfig() map[string]any {
	return map[string]any{
		"name":                t.Name(),
		"path":                t.dir,
		"retry_count":         100,
		"retry_delay_initial": t.generationDelay.String(),
		"retry_delay_max":     t.generationDelay.String(),
		"command": []string{
			"/usr/bin/env",
			"GENERATION_FILE=" + t.generation.path,
			"STATUS_DIR=" + t.statusDir,
			"/bin/sh", "-x", "-e", "-c", `
read -r generation _ < "${GENERATION_FILE:?}"
read -r target _ < "${BAAMHACKL_INPUT:?}"

if test \( -z "${generation}" \) -o \( -z "${target}" \) || test "${generation}" -lt "${target}"; then
	echo "Not yet ready (generation ${generation}, target ${target})." >&2
	exit 1
fi

name=$(basename -- "${BAAMHACKL_INPUT:?}")

checkmark="${STATUS_DIR:?}/checkmark_${name}"

if test -e "${checkmark}"; then
	echo "File ${checkmark} exists already." >&2
	exit 1
fi

: > "${checkmark}"

exit 0
`,
		},
	}
}

func (t *multifileTest) newFile(targetGeneration int) (*file, error) {
	t.counter++

	name := fmt.Sprintf("%x", t.counter)

	f := &file{
		name:          name,
		inputPath:     filepath.Join(t.dir, name),
		checkmarkPath: filepath.Join(t.statusDir, "checkmark_"+name),
	}

	return f, writeInt(f.inputPath, targetGeneration)
}

func (t *multifileTest) addTestFile(targetGeneration int) error {
	if p, err := t.newFile(targetGeneration); err != nil {
		return err
	} else {
		t.pending[p] = struct{}{}
	}

	return nil
}

func (t *multifileTest) Setup() error {
	return t.addTestFile(3)
}

func (t *multifileTest) deleteFinished() ([]string, error) {
	var remaining []string

	for p := range t.pending {
		ok, err := p.finished()

		switch {
		case err != nil:
			return nil, err
		case ok:
			delete(t.pending, p)
		default:
			remaining = append(remaining, p.name)
		}
	}

	return remaining, nil
}

func (t *multifileTest) Run(ctx context.Context) error {
	logger := zap.L()

	for i := 0; i < 10; i++ {
		if err := t.addTestFile(i); err != nil {
			return err
		}
	}

	logger.Info("Waiting for files to be processed")

	for {
		remaining, err := t.deleteFinished()
		if err != nil {
			return err
		}

		if len(remaining) == 0 {
			break
		}

		logger.Info("Tasks not yet finished", zap.Strings("names", remaining))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(t.generationDelay):
		}

		if err := t.generation.increment(); err != nil {
			return err
		}

		logger.Info("Generation incremented", zap.Int("count", t.generation.count))
	}

	return nil
}
