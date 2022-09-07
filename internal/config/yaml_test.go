package config

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type marshalTestSlice []marshalTest

func (ts marshalTestSlice) run(t *testing.T) {
	t.Helper()

	for _, tc := range ts {
		tc.run(t)
	}
}

type marshalTest struct {
	name    string
	input   string
	want    interface{}
	wantErr *regexp.Regexp
}

func (tc *marshalTest) run(t *testing.T) {
	t.Helper()
	t.Run(tc.name, func(t *testing.T) {
		wantType := reflect.ValueOf(tc.want).Type()

		got := reflect.New(wantType)

		if tc.wantErr != nil {
			zero := reflect.Zero(wantType)

			if diff := cmp.Diff(tc.want, zero.Interface()); diff != "" {
				t.Errorf("When expecting an error (%q), the wanted value must be the zero value. Diff (-want +got):\n%s", tc.wantErr, diff)
			}
		}

		if err := validatedUnmarshal(strings.NewReader(tc.input), got.Interface()); err != nil {
			if tc.wantErr == nil {
				t.Errorf("validatedUnmarshal(%q) failed: %v", tc.input, err)
			} else if !tc.wantErr.MatchString(err.Error()) {
				t.Errorf("validatedUnmarshal(%q) failed with %q, want error match for %q", tc.input, err, tc.wantErr.String())
			}
		} else if tc.wantErr != nil {
			t.Errorf("validatedUnmarshal() succeeded, want error match for %q", tc.wantErr.String())
		} else {
			if diff := cmp.Diff(tc.want, got.Elem().Interface()); diff != "" {
				t.Errorf("Decoded diff (-want +got):\n%s", diff)
			}

			var buf strings.Builder

			if err := marshal(&buf, got.Interface()); err != nil {
				t.Errorf("marshal() failed: %v", err)
			} else {
				got2 := reflect.New(wantType)

				if err := validatedUnmarshal(strings.NewReader(buf.String()), got2.Interface()); err != nil {
					t.Errorf("validatedUnmarshal(%q) after marshalling failed: %v", buf.String(), err)
				}
			}
		}
	})
}
