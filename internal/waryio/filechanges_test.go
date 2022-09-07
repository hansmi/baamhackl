package waryio

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
)

func TestDescribeChanges(t *testing.T) {
	tmpdir := t.TempDir()

	sizePath := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "size"), "")
	sizeBefore := testutil.MustLstat(t, sizePath)
	testutil.MustWriteFile(t, sizePath, "changed")
	sizeAfter := testutil.MustLstat(t, sizePath)

	mtimePath := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "mtime"), "")
	mtimeBefore := testutil.MustLstat(t, mtimePath)
	if err := os.Chtimes(mtimePath, time.Time{}, time.Date(2006, time.February, 1, 3, 4, 5, 0, time.UTC)); err != nil {
		t.Errorf("Chtimes() failed: %v", err)
	}
	mtimeAfter := testutil.MustLstat(t, mtimePath)

	typePath := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "type"), "")
	typeBefore := testutil.MustLstat(t, typePath)
	typeAfter := testutil.MustLstat(t, t.TempDir())

	type testCase struct {
		name string
		prev os.FileInfo
		cur  os.FileInfo
		want *regexp.Regexp
	}

	cases := []testCase{
		{
			name: "same dir",
			prev: testutil.MustLstat(t, tmpdir),
			cur:  testutil.MustLstat(t, tmpdir),
		},
		{
			name: "different dir",
			prev: testutil.MustLstat(t, tmpdir),
			cur:  testutil.MustLstat(t, t.TempDir()),
			want: regexp.MustCompile(`^moved or replaced \(not the same file\)$`),
		},
		{
			name: "changed size",
			prev: sizeBefore,
			cur:  sizeAfter,
			want: regexp.MustCompile(`(?i)^size changed \(\d+\s*!=\s*\d+\)$`),
		},
		{
			name: "changed mtime",
			prev: mtimeBefore,
			cur:  mtimeAfter,
			want: regexp.MustCompile(`(?i)^modification time changed\b`),
		},
		{
			name: "changed type",
			prev: typeBefore,
			cur:  typeAfter,
			want: regexp.MustCompile(`(?i)^type changed\b`),
		},
	}

	for _, i := range []os.FileInfo{
		sizeBefore,
		sizeAfter,
		mtimeBefore,
		mtimeAfter,
		typeBefore,
		typeAfter,
	} {
		cases = append(cases, testCase{
			name: fmt.Sprintf("same file for %s", i.Name()),
			prev: i,
			cur:  i,
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DescribeChanges(tc.prev, tc.cur)

			if tc.want != nil {
				found := false
				for _, i := range got {
					if found = tc.want.MatchString(i); found {
						break
					}
				}
				if !found {
					t.Errorf("DescribeChanges() result contains no match for %q: %q", tc.want.String(), got)
				}
			} else if diff := cmp.Diff(FileChanges{}, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("DescribeChanges() result diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want == nil, got.Empty()); diff != "" {
				t.Errorf("Empty() result diff (-want +got):\n%s", diff)
			}
		})
	}
}
