package uniquename

import (
	"fmt"

	"github.com/rivo/uniseg"
)

// combineWithMaxLen produces a string containing a maximum of maxBytes bytes
// in UTF-8 encoding. First the prefix is cut off at the end and if that
// doesn't suffice, the suffix from the beginning. The middle part must always
// fit.
func combineWithMaxLen(prefix, middle, suffix string, maxBytes int) string {
	if len(middle) > maxBytes {
		panic(fmt.Sprintf("middle part %q longer than %d bytes", middle, maxBytes))
	}

	var cur string

	for g := uniseg.NewGraphemes(prefix); g.Next(); {
		_, endBytePos := g.Positions()

		if endBytePos+len(middle)+len(suffix) > maxBytes {
			break
		}

		cur = prefix[:endBytePos]
	}

	prefix = cur

	cur = ""

	for g := uniseg.NewGraphemes(suffix); g.Next(); {
		startBytePos, _ := g.Positions()

		if len(prefix)+len(middle)+(len(suffix)-startBytePos) <= maxBytes {
			cur = suffix[startBytePos:]
			break
		}
	}

	suffix = cur

	return prefix + middle + suffix
}
