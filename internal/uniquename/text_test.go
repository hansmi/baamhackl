package uniquename

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
)

func TestCombineWithMaxLen(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix string
		middle string
		suffix string
		want   map[int]string
	}{
		{name: "empty"},
		{
			name:   "prefix only",
			prefix: "_prefix_",
			want: map[int]string{
				10: "_prefix_",
			},
		},
		{
			name:   "middle only",
			middle: "_middle_",
			want: map[int]string{
				10: "_middle_",
			},
		},
		{
			name:   "suffix only",
			suffix: "_suffix_",
			want: map[int]string{
				10: "_suffix_",
			},
		},
		{
			name:   "prefix fully trimmed",
			prefix: "_prefix_",
			middle: "_middle_",
			suffix: "01234",
			want: map[int]string{
				len("_middle_01234"): "_middle_01234",
			},
		},
		{
			name:   "prefix partially trimmed",
			prefix: ".prefix.",
			middle: "_middle_",
			suffix: "01234",
			want: map[int]string{
				len(".pre_middle_01234"): ".pre_middle_01234",
			},
		},
		{
			name:   "prefix fully, suffix partially trimmed",
			prefix: ".prefix.",
			middle: "_middle_",
			suffix: "|suffix|",
			want: map[int]string{
				len("_middle_fix|"): "_middle_fix|",
			},
		},
		{
			name:   "trimmed umlaut in prefix",
			prefix: "Ka\u0308se",
			want: map[int]string{
				0: "",
				3: "K",
				4: "Ka\u0308",
				5: "Ka\u0308s",
			},
		},
		{
			name:   "trimmed umlaut in suffix",
			suffix: "Ka\u0308se",
			want: map[int]string{
				0:  "",
				3:  "se",
				5:  "a\u0308se",
				10: "Ka\u0308se",
			},
		},
		{
			name:   "complex name",
			prefix: "DE=\U0001f1e9\U0001f1ea,AU=\U0001f1e6\U0001f1fa",
			middle: "(01234)",
			suffix: "NL=\U0001f1f3\U0001f1f1,IN=\U0001f1ee\U0001f1f3",
			want: map[int]string{
				7:  "(01234)",
				18: "(01234)IN=\U0001f1ee\U0001f1f3",
				30: "(01234)NL=\U0001f1f3\U0001f1f1,IN=\U0001f1ee\U0001f1f3",
				34: "DE=(01234)NL=\U0001f1f3\U0001f1f1,IN=\U0001f1ee\U0001f1f3",
				49: "DE=\U0001f1e9\U0001f1ea,AU=(01234)NL=\U0001f1f3\U0001f1f1,IN=\U0001f1ee\U0001f1f3",
				100: ("DE=\U0001f1e9\U0001f1ea,AU=\U0001f1e6\U0001f1fa" +
					"(01234)" +
					"NL=\U0001f1f3\U0001f1f1,IN=\U0001f1ee\U0001f1f3"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for maxLen, want := range tc.want {
				t.Run(fmt.Sprint(maxLen), func(t *testing.T) {
					got := combineWithMaxLen(tc.prefix, tc.middle, tc.suffix, maxLen)

					if diff := cmp.Diff(want, got); diff != "" {
						t.Errorf("combineWithMaxLen() diff (-want +got):\n%s", diff)
					}

					if len(got) > maxLen {
						t.Errorf("combineWithMaxLen() returned string longer than %d bytes: %q", maxLen, got)
					}

					if !utf8.ValidString(got) {
						t.Errorf("combineWithMaxLen() returned invalid UTF-8: %q", got)
					}
				})
			}
		})
	}
}
