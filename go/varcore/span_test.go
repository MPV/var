package varcore

import "testing"

func TestUTF16LenCountsAstralAsTwo(t *testing.T) {
	if got := utf16Len("a😀b"); got != 4 {
		t.Errorf("utf16Len = %d, want 4", got)
	}
	if got := utf16Len("café"); got != 4 {
		t.Errorf("utf16Len = %d, want 4", got)
	}
}

func TestUTF16SliceRespectsCodeUnits(t *testing.T) {
	// "😀" is 2 code units [0,2); "x" starts at unit 2.
	if got := utf16Slice("😀x", 2, 3); got != "x" {
		t.Errorf("utf16Slice = %q, want %q", got, "x")
	}
	if got := utf16Slice("😀x", 0, 2); got != "😀" {
		t.Errorf("utf16Slice = %q, want %q", got, "😀")
	}
}

func TestLineColInUTF16Units(t *testing.T) {
	src := "ab\n😀c"
	// offset 0 -> line 1 col 1
	if l, c := lineCol(src, 0); l != 1 || c != 1 {
		t.Errorf("lineCol(0) = %d,%d want 1,1", l, c)
	}
	// after the newline, before the emoji: offset 3 -> line 2 col 1
	if l, c := lineCol(src, 3); l != 2 || c != 1 {
		t.Errorf("lineCol(3) = %d,%d want 2,1", l, c)
	}
	// after the emoji (2 units): offset 5 -> line 2 col 3
	if l, c := lineCol(src, 5); l != 2 || c != 3 {
		t.Errorf("lineCol(5) = %d,%d want 2,3", l, c)
	}
}

func TestSpanFromOffsets(t *testing.T) {
	src := "hello\nworld"
	got := spanFromOffsets(src, 6, 11)
	want := Span{StartOffset: 6, EndOffset: 11, StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 6}
	if got != want {
		t.Errorf("spanFromOffsets = %+v, want %+v", got, want)
	}
}
