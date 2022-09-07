// Package teelog implements a wrapper sending log messages to a parent logger
// as well as a file with newline-delimited JSON fragments (NDJSON).
package teelog

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/unix"
)

type File struct {
	Parent *zap.Logger

	// Path to log file.
	Path string

	fs interface {
		OpenFile(name string, flag int, perm os.FileMode) (afero.File, error)
	}
	encoder zapcore.Encoder
}

func (f *File) open() (*zap.Logger, func() error, error) {
	if f.fs == nil {
		f.fs = afero.NewOsFs()
	}

	if f.Parent == nil {
		f.Parent = zap.NewNop()
	}

	if f.encoder == nil {
		f.encoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			MessageKey:     "msg",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		})
	}

	fh, err := f.fs.OpenFile(f.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return nil, nil, err
	}

	logger := zap.New(zapcore.NewTee(
		f.Parent.Core(),
		zapcore.NewCore(f.encoder, zapcore.Lock(fh), zapcore.DebugLevel),
	))

	return logger, func() error {
		syncErr := logger.Sync()

		if errors.Is(syncErr, unix.EINVAL) {
			syncErr = nil
		}

		return multierr.Combine(syncErr, fh.Close())
	}, nil
}

func (f *File) Wrap(fn func(*zap.Logger) error) (err error) {
	logger, logClose, err := f.open()
	if err != nil {
		return fmt.Errorf("log setup failed: %w", err)
	}

	defer multierr.AppendInvoke(&err, multierr.Invoke(logClose))

	return fn(logger)
}
