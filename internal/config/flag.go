package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const PathEnvVar = "BAAMHACKL_CONFIG_FILE"

var ErrMissingFile = errors.New("missing configuration file")

func read(cfg *Root, path string) error {
	fh, err := os.Open(path)
	if err != nil {
		return err
	}

	defer fh.Close()

	return cfg.Unmarshal(fh)
}

// Flag defines a command line flag to load a configuration file.
type Flag struct {
	path string
}

func (f *Flag) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.path, "config", os.Getenv(PathEnvVar),
		"Path to configuration file (defaults to "+PathEnvVar+" environment variable).")
}

func (f *Flag) Load() (*Root, error) {
	if f.path == "" {
		return nil, ErrMissingFile
	}

	cfg := &Root{}

	if err := read(cfg, f.path); err != nil {
		return nil, fmt.Errorf("loading configuration from %q failed: %w", f.path, err)
	}

	return cfg, nil
}
