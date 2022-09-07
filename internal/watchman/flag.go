package watchman

import (
	"flag"
)

type Flags struct {
	program string
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.program, "watchman_program", "watchman",
		"Watchman executable. Looked up via PATH if not given as an absolute path.")
}

func (f *Flags) Args() []string {
	return []string{f.program}
}

func (f *Flags) NewClient() *CommandClient {
	return NewCommandClient(f.Args())
}
