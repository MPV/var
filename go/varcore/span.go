package varcore

// Span is a source position, the position primitive of the pipeline. Port of
// span.py / span.ts. All offsets and columns are measured in UTF-16 code units
// (an astral character such as 😀 counts as 2), matching the JavaScript
// reference and LSP's default position encoding — the goldens are generated in
// this encoding. Go strings are UTF-8, so the helpers below convert on the fly.
type Span struct {
	StartOffset int
	EndOffset   int
	StartLine   int
	StartCol    int
	EndLine     int
	EndCol      int
}

// utf16Len returns the number of UTF-16 code units in s.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// byteIndexForUTF16 returns the byte index into source at the given UTF-16 code
// unit offset. Used to splice the UTF-8 source by UTF-16 offsets. If u16 lands
// in the middle of a surrogate pair (which never happens for a well-formed
// offset) it rounds up past that rune.
func byteIndexForUTF16(source string, u16 int) int {
	count := 0
	for i, r := range source {
		if count >= u16 {
			return i
		}
		if r > 0xFFFF {
			count += 2
		} else {
			count++
		}
	}
	return len(source)
}

// utf16Slice returns the substring of source between the two UTF-16 offsets.
func utf16Slice(source string, startU16, endU16 int) string {
	a := byteIndexForUTF16(source, startU16)
	b := byteIndexForUTF16(source, endU16)
	return source[a:b]
}

// lineCol returns the 1-based line and column (column in UTF-16 code units) at
// the given UTF-16 offset. Port of line_col in span.py.
func lineCol(source string, offsetU16 int) (line, col int) {
	line, col = 1, 1
	count := 0
	for _, r := range source {
		if count >= offsetU16 {
			break
		}
		if r == '\n' {
			line, col = line+1, 1
		} else if r > 0xFFFF {
			col += 2
		} else {
			col++
		}
		if r > 0xFFFF {
			count += 2
		} else {
			count++
		}
	}
	return line, col
}

// spanFromOffsets builds a Span from a pair of UTF-16 offsets, computing the
// line/column endpoints. Port of span_from_offsets in span.py.
func spanFromOffsets(source string, startU16, endU16 int) Span {
	sl, sc := lineCol(source, startU16)
	el, ec := lineCol(source, endU16)
	return Span{
		StartOffset: startU16,
		EndOffset:   endU16,
		StartLine:   sl,
		StartCol:    sc,
		EndLine:     el,
		EndCol:      ec,
	}
}
