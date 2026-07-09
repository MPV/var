package varcore

import (
	"strings"
	"unicode"
)

// parseRowCells splits a `| a | b |` table row into trimmed cells and per-cell
// spans. Port of parse_row_cells in table_cells.py / table-cells.ts.
//
// lineStartOffset is the UTF-16 offset of the first character of lineText within
// source. All emitted span offsets are UTF-16 code units. Go strings are byte
// indexed, so pipe positions found by byte index are converted to UTF-16 with
// utf16Len over the preceding bytes.
func parseRowCells(lineText string, lineStartOffset int, source string) ([]string, []Span) {
	firstByte := strings.IndexByte(lineText, '|')
	lastByte := strings.LastIndexByte(lineText, '|')
	if firstByte < 0 || lastByte <= firstByte {
		return nil, nil
	}

	// UTF-16 offset within lineText of the first '|'. '|' is ASCII (1 unit), so
	// the char after it starts one unit later.
	firstU16 := utf16Len(lineText[:firstByte])
	innerStartU16 := firstU16 + 1

	inner := lineText[firstByte+1 : lastByte]

	cells := make([]string, 0)
	cellSpans := make([]Span, 0)
	cursor := 0 // running UTF-16 position within inner

	for _, seg := range strings.Split(inner, "|") {
		trimmed := strings.TrimSpace(seg)
		leading := utf16Len(seg) - utf16Len(strings.TrimLeftFunc(seg, unicode.IsSpace))
		absStart := lineStartOffset + innerStartU16 + cursor + leading
		cells = append(cells, trimmed)
		cellSpans = append(cellSpans, spanFromOffsets(source, absStart, absStart+utf16Len(trimmed)))
		cursor += utf16Len(seg) + 1 // +1 for the '|' delimiter
	}

	return cells, cellSpans
}
