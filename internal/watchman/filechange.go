package watchman

import (
	"encoding/json"
	"time"
)

var FileChangeFields = []string{"name", "size", "mtime_us", "cclock"}

type usecTimeWrapper struct {
	*time.Time
}

func (t usecTimeWrapper) MarshalJSON() ([]byte, error) {
	if t.Time == nil {
		return []byte("null"), nil
	}

	return json.Marshal(t.Time.UnixMicro())
}

func (t *usecTimeWrapper) UnmarshalJSON(data []byte) error {
	var usec int64

	if err := json.Unmarshal(data, &usec); err != nil {
		return err
	}

	*t.Time = time.UnixMicro(usec)

	return nil
}

type FileChange struct {
	Name   string    `json:"name"`
	Size   int64     `json:"size"`
	MTime  time.Time `json:"-"`
	CClock string    `json:"cclock"`
}

type plainFileChange FileChange

type rawFileChange struct {
	plainFileChange
	MTimeWrapper usecTimeWrapper `json:"mtime_us"`
}

func (c FileChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(rawFileChange{
		plainFileChange: plainFileChange(c),
		MTimeWrapper:    usecTimeWrapper{&c.MTime},
	})
}

func (c *FileChange) UnmarshalJSON(data []byte) error {
	var r rawFileChange

	r.MTimeWrapper.Time = &r.plainFileChange.MTime

	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	*c = FileChange(r.plainFileChange)

	return nil
}
