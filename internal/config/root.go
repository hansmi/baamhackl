package config

import (
	"io"
)

type Root struct {
	Handlers []*Handler `yaml:"handlers" validate:"unique=Name"`
}

func (r *Root) Unmarshal(reader io.Reader) error {
	// Reset to zero values
	*r = Root{}

	return validatedUnmarshal(reader, r)
}

func (r *Root) Marshal(w io.Writer) error {
	return marshal(w, r)
}
