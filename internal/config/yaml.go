package config

import (
	"errors"
	"io"

	"github.com/goccy/go-yaml"
)

var ErrMultipleFragments = errors.New("input contained multiple YAML fragments")

func validatedUnmarshal(r io.Reader, v any) error {
	opts := []yaml.DecodeOption{
		yaml.Strict(),
		yaml.Validator(customValidate.get()),
	}

	dec := yaml.NewDecoder(r, opts...)

	if err := dec.Decode(v); !(err == nil || errors.Is(err, io.EOF)) {
		return err
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrMultipleFragments
	}

	return nil
}

func marshal(w io.Writer, v any) error {
	opts := []yaml.EncodeOption{
		yaml.Flow(false),
		yaml.Indent(2),
		yaml.IndentSequence(true),
	}

	enc := yaml.NewEncoder(w, opts...)

	if err := enc.Encode(v); err != nil {
		return err
	}

	return enc.Close()
}
