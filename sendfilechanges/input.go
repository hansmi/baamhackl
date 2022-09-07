package sendfilechanges

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hansmi/baamhackl/internal/watchman"
)

func readInput(r io.Reader) ([]watchman.FileChange, error) {
	var changes []watchman.FileChange

	dec := json.NewDecoder(r)

	if err := dec.Decode(&changes); err != nil {
		return nil, fmt.Errorf("reading input failed: %w", err)
	}

	return changes, nil
}
