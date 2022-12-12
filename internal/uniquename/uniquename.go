package uniquename

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/sys/unix"
)

var suffixRe = regexp.MustCompile(`\s+\([0-9a-fA-F]+\)\s*`)

var ErrMissingTime = errors.New("missing timestamp")

type Options struct {
	// prepend current time to name, even for original name
	TimePrefixEnabled          bool
	TimePrefixLayout           string
	TimePrefixSupportedLayouts []string

	// if name contains an extension
	BeforeExtension    bool
	MaxExtensionLength int
}

var DefaultOptions = Options{
	TimePrefixEnabled: true,
	TimePrefixLayout:  "2006-01-02T150405",
	TimePrefixSupportedLayouts: []string{
		"2006-01-02T150405-0700",
		"2006-01-02T150405",
	},
	BeforeExtension:    true,
	MaxExtensionLength: 10,
}

type Generator struct {
	dir, timePrefix, prefix, suffix string

	originalName string
	timeSuffix   time.Time
}

func New(path string, opts Options) (*Generator, error) {
	g := &Generator{}
	g.dir, g.prefix = filepath.Split(path)

	if g.prefix == "" {
		return nil, fmt.Errorf("%w: path %q contains no name", os.ErrInvalid, path)
	}

	g.originalName = g.prefix

	if opts.BeforeExtension && opts.MaxExtensionLength > 0 {
		if ext := filepath.Ext(g.prefix); len(ext) < opts.MaxExtensionLength {
			base := g.prefix[:len(g.prefix)-len(ext)]

			if base == "" {
				base = ext
				ext = ""
			}

			g.prefix = base
			g.suffix = ext
		}
	}

	g.prefix = suffixRe.ReplaceAllLiteralString(g.prefix, "")

	if now := globalRuntime.Clock.Now().In(globalRuntime.Loc); opts.TimePrefixEnabled {
		formatted := now.Format(opts.TimePrefixLayout)
		g.timePrefix = strings.TrimRightFunc(formatted, unicode.IsSpace) + " "
	} else {
		g.timeSuffix = now
	}

	return g, nil
}

func (g *Generator) Next() (path string, ok bool) {
	var name string

	if g.originalName != "" {
		name = g.timePrefix + g.originalName
		g.originalName = ""
	} else {
		var uniq string

		if g.timeSuffix.IsZero() {
			uniq = strconv.FormatInt(int64(globalRuntime.RandInt()), 16)
		} else {
			uniq = g.timeSuffix.Format("20060102150405")
			g.timeSuffix = time.Time{}
		}

		name = combineWithMaxLen(g.timePrefix+g.prefix, " ("+uniq+")", g.suffix, unix.NAME_MAX)
	}

	return filepath.Join(g.dir, name), true
}

func ExtractTime(path string, opts Options) (time.Time, error) {
	_, name := filepath.Split(path)
	name = strings.TrimLeftFunc(name, unicode.IsSpace)

	if pos := strings.IndexFunc(name, unicode.IsSpace); pos > 0 {
		name = name[:pos]
	}

	for _, layout := range opts.TimePrefixSupportedLayouts {
		if ts, err := time.ParseInLocation(layout, name, globalRuntime.Loc); err == nil {
			return ts, nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: %s", ErrMissingTime, path)
}
