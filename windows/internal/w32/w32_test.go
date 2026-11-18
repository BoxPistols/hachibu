//go:build windows

package w32

import "testing"

func TestUTF16SliceDropsNULAndTerminates(t *testing.T) {
	for in, want := range map[string]int{"": 1, "ab": 3, "a\x00b": 3, "\x00": 1, "日本": 3} {
		got := UTF16Slice(in)
		if len(got) != want || got[len(got)-1] != 0 {
			t.Errorf("%q → %v", in, got)
		}
	}
}
