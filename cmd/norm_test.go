package cmd

import "testing"

func TestNormalizeCodeConvertsFullWidthAndCJKPunctuation(t *testing.T) {
	input := "if\uff08a\uff1d\uff11\uff0cb\uff1d\u201c\u4f60\u597d\u3001\u4e16\u754c\u3002\u201d\uff09\u3000\u3010ok\u3011 \u300aT\u300b"
	want := "if(a=1,b=\"\u4f60\u597d,\u4e16\u754c.\") [ok] <T>"

	if got := normalizeCode(input); got != want {
		t.Fatalf("normalizeCode() = %q, want %q", got, want)
	}
}
