package config

import (
	"regexp"
	"testing"
)

func TestRoot(t *testing.T) {
	marshalTestSlice{
		{
			name:  "empty",
			input: `{}`,
			want:  Root{},
		},
		{
			name: "one",
			input: `
---
handlers:
- name: aaa
  path: /test/dir
  command: ["path", "to", "command"]
`,
			want: Root{
				Handlers: []*Handler{
					func() *Handler {
						o := HandlerDefaults
						o.Name = "aaa"
						o.Path = "/test/dir"
						o.Command = []string{"path", "to", "command"}
						return &o
					}(),
				},
			},
		},
		{
			name: "duplicate name",
			input: `
---
handlers:
- name: aaa
  path: /test/first
  command: ['x']
- name: aaa
  path: /test/second
  command: ['y']
`,
			want:    Root{},
			wantErr: regexp.MustCompile(`(?i)\bvalidation\b.*\bfailed\b.*\bunique\b`),
		},
	}.run(t)
}
