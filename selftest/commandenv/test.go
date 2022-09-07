package commandenv

import (
	"bufio"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/multierr"
)

const commandEnvLineCount = 1024 * 1024

type commandEnvTest struct {
	dir        string
	targetDir  string
	wantDigest []byte
}

func New(dir string) (*commandEnvTest, error) {
	var err error

	t := &commandEnvTest{}

	if t.dir, err = os.MkdirTemp(dir, "commandenv*"); err != nil {
		return nil, err
	}

	if t.targetDir, err = os.MkdirTemp(dir, "commandtarget*"); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *commandEnvTest) Name() string {
	return "command environment"
}

func (t *commandEnvTest) HandlerConfig() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"path":        t.dir,
		"retry_count": 0,
		"command": []string{
			"/usr/bin/env", "TARGET_DIR=" + t.targetDir,
			"/bin/sh", "-x", "-e", "-c", `
if ! test . -ef "${BAAMHACKL_WORKDIR:?}"; then
	echo "Current directory is ${PWD}, not ${BAAMHACKL_WORKDIR}" >&2
	exit 1
fi

cp -v "${BAAMHACKL_INPUT:?}" done.txt

"${BAAMHACKL_PROGRAM:?}" move-into "${TARGET_DIR:?}" done.txt
`},
	}
}

func (t *commandEnvTest) Setup() (err error) {
	fh, err := os.Create(filepath.Join(t.dir, "trigger"))
	if err != nil {
		return err
	}

	defer multierr.AppendInvoke(&err, multierr.Close(fh))

	bfh := bufio.NewWriterSize(fh, 128*1024)

	digest := fnv.New128a()
	w := io.MultiWriter(bfh, digest)

	for i := 0; i < commandEnvLineCount; i++ {
		if _, err := fmt.Fprintf(w, "%d\n", i); err != nil {
			return err
		}
	}

	if err := bfh.Flush(); err != nil {
		return err
	}

	t.wantDigest = digest.Sum(nil)

	return nil
}

func (t *commandEnvTest) Run(ctx context.Context) error {
	b := backoff.NewExponentialBackOff()
	b.RandomizationFactor = 0.1
	b.MaxInterval = time.Second
	b.MaxElapsedTime = 10 * time.Second

	path := filepath.Join(t.targetDir, "done.txt")

	return backoff.Retry(func() (err error) {
		fh, err := os.Open(path)
		if err != nil {
			return err
		}

		defer multierr.AppendInvoke(&err, multierr.Close(fh))

		digest := fnv.New128a()

		if _, err := io.Copy(digest, bufio.NewReader(fh)); err != nil {
			return err
		}

		if diff := cmp.Diff(t.wantDigest, digest.Sum(nil)); diff != "" {
			return fmt.Errorf("digest of copied file differs (-want +got):\n%s", diff)
		}

		return nil
	}, backoff.WithContext(b, ctx))
}
