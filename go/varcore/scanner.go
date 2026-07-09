package varcore

import (
	"regexp"
	"strings"
	"unicode"
)

// scanner.go — markdown block scanner. Port of scanner.py / scanner.ts.
//
// UTF-16 rule: all offsets (rawLine offsets and Span offsets) count UTF-16 code
// units. Go strings are UTF-8/byte indexed, so where the reference uses a
// code-point index (str.find / indexOf) this port finds a byte index and
// converts the preceding bytes to a UTF-16 length via utf16Len.

// rawLine is one source line with its UTF-16 start/end offsets (end excludes the
// trailing newline).
type rawLine struct {
	text        string
	startOffset int
	endOffset   int
}

// ScannerPlugin participates in block recognition before the built-in rules.
// The core conformance corpus is reproduced with no plugins; plugins are a
// config/LSP concern.
type ScannerPlugin interface {
	// TryScan returns a block and the next line index to resume at, or false.
	TryScan(source string, lines []rawLine, startIdx int) (Block, int, bool)
}

// Regexes — verbatim ports of the reference constants, except THEMATIC (which
// uses a backreference RE2 cannot express and is hand-checked in isThematicBreak).
var (
	ulRe          = regexp.MustCompile(`^(\s*)([-*+])\s+(.*)$`)
	olRe          = regexp.MustCompile(`^(\s*)(\d+)([.)])\s+(.*)$`)
	bqRe          = regexp.MustCompile(`^>\s?(.*)$`)
	fenceRe       = regexp.MustCompile("^(`{3,})\\s*(\\S*)\\s*$")
	rowRe         = regexp.MustCompile(`^\|(.+)\|\s*$`)
	delimRe       = regexp.MustCompile(`^\|\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|\s*$`)
	headingRe     = regexp.MustCompile(`^(#{1,6})\s+(.*?)(?:\s+#+)?\s*$`)
	headingStart  = regexp.MustCompile(`^#{1,6}\s+`)
	blankParaStop = regexp.MustCompile(`\n\s*\n`)
)

// isThematicBreak reports whether text is a thematic break: 3+ of the same
// character from {-,*,_}, separated/surrounded only by whitespace. Hand-ports
// THEMATIC_RE `^\s*([-*_])(\s*\1){2,}\s*$` (RE2 has no backreferences).
func isThematicBreak(text string) bool {
	var c rune
	count := 0
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if c == 0 {
			if r != '-' && r != '*' && r != '_' {
				return false
			}
			c = r
		} else if r != c {
			return false
		}
		count++
	}
	return count >= 3
}

// Scan scans source into an immutable sequence of Block nodes.
func Scan(source string, plugins []ScannerPlugin) []Block {
	blocks := make([]Block, 0)
	lines := splitLines(source)

	i := 0
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line.text) == "" {
			i++
			continue
		}

		if b, next, ok := runPlugins(source, lines, i, plugins); ok {
			blocks = append(blocks, b)
			i = next
			continue
		}
		if b, next, ok := tryFence(source, lines, i); ok {
			blocks = append(blocks, b)
			i = next
			continue
		}
		if b, next, ok := tryTable(source, lines, i); ok {
			blocks = append(blocks, b)
			i = next
			continue
		}
		if b, ok := tryThematic(source, line); ok {
			blocks = append(blocks, b)
			i++
			continue
		}
		if b, next, ok := tryBlockquote(source, lines, i); ok {
			blocks = append(blocks, b)
			i = next
			continue
		}
		if b, ok := tryHeading(source, line); ok {
			blocks = append(blocks, b)
			i++
			continue
		}
		if b, ok := tryListItem(source, line); ok {
			blocks = append(blocks, b)
			i++
			continue
		}

		paragraph, next := consumeParagraph(source, lines, i, plugins)
		blocks = append(blocks, paragraph)
		i = next
	}

	return blocks
}

func runPlugins(source string, lines []rawLine, startIdx int, plugins []ScannerPlugin) (Block, int, bool) {
	for _, p := range plugins {
		if b, next, ok := p.TryScan(source, lines, startIdx); ok {
			return b, next, true
		}
	}
	return nil, 0, false
}

// splitLines splits source into rawLines with UTF-16 start/end offsets. Iterates
// runes tracking both a byte index (for Go slicing) and a UTF-16 offset. '\n' is
// U+000A (1 UTF-16 unit, 1 byte).
func splitLines(source string) []rawLine {
	out := make([]rawLine, 0)
	startU16 := 0
	currentU16 := 0
	startByte := 0

	for i, r := range source {
		if r == '\n' {
			out = append(out, rawLine{
				text:        source[startByte:i],
				startOffset: startU16,
				endOffset:   currentU16,
			})
			startU16 = currentU16 + 1
			startByte = i + 1
		}
		if r > 0xFFFF {
			currentU16 += 2
		} else {
			currentU16++
		}
	}

	out = append(out, rawLine{
		text:        source[startByte:],
		startOffset: startU16,
		endOffset:   currentU16,
	})
	return out
}

func tryThematic(source string, line rawLine) (Block, bool) {
	if !isThematicBreak(line.text) {
		return nil, false
	}
	return ThematicBreak{Sp: spanFromOffsets(source, line.startOffset, line.endOffset)}, true
}

func tryHeading(source string, line rawLine) (Block, bool) {
	m := headingRe.FindStringSubmatch(line.text)
	if m == nil {
		return nil, false
	}
	level := len(m[1])
	text := strings.TrimSpace(m[2])
	return Heading{
		Level: level,
		Text:  text,
		Sp:    spanFromOffsets(source, line.startOffset, line.endOffset),
	}, true
}

func tryListItem(source string, line rawLine) (Block, bool) {
	if ul := ulRe.FindStringSubmatch(line.text); ul != nil {
		text := ul[3]
		markerStart := line.startOffset + utf16Len(ul[1])
		markerEnd := markerStart + utf16Len(ul[2])
		byteIdx := strings.Index(line.text, text)
		textStart := line.startOffset + utf16Len(line.text[:byteIdx])
		return ListItem{
			Text:       text,
			Sp:         spanFromOffsets(source, line.startOffset, line.endOffset),
			SegmentMap: []SegmentOffset{{TextOffset: 0, SourceOffset: textStart}},
			Ordered:    false,
			MarkerSpan: spanFromOffsets(source, markerStart, markerEnd),
		}, true
	}
	if ol := olRe.FindStringSubmatch(line.text); ol != nil {
		text := ol[4]
		markerStart := line.startOffset + utf16Len(ol[1])
		markerEnd := markerStart + utf16Len(ol[2]) + utf16Len(ol[3])
		byteIdx := strings.Index(line.text, text)
		textStart := line.startOffset + utf16Len(line.text[:byteIdx])
		return ListItem{
			Text:       text,
			Sp:         spanFromOffsets(source, line.startOffset, line.endOffset),
			SegmentMap: []SegmentOffset{{TextOffset: 0, SourceOffset: textStart}},
			Ordered:    true,
			MarkerSpan: spanFromOffsets(source, markerStart, markerEnd),
		}, true
	}
	return nil, false
}

func tryBlockquote(source string, lines []rawLine, startIdx int) (Block, int, bool) {
	if startIdx >= len(lines) {
		return nil, 0, false
	}
	first := lines[startIdx]
	m := bqRe.FindStringSubmatch(first.text)
	if m == nil {
		return nil, 0, false
	}

	firstSegment := m[1]
	byteIdx := strings.Index(first.text, firstSegment)
	segments := []string{firstSegment}
	segmentMap := []SegmentOffset{{
		TextOffset:   0,
		SourceOffset: first.startOffset + utf16Len(first.text[:byteIdx]),
	}}
	joinedTextOffset := utf16Len(firstSegment)

	i := startIdx + 1
	endOffset := first.endOffset
	for i < len(lines) {
		ln := lines[i]
		nextM := bqRe.FindStringSubmatch(ln.text)
		if nextM == nil {
			break
		}
		segment := nextM[1]
		byteIdx2 := strings.Index(ln.text, segment)
		joinedTextOffset++ // newline separator (1 UTF-16 unit)
		segmentMap = append(segmentMap, SegmentOffset{
			TextOffset:   joinedTextOffset,
			SourceOffset: ln.startOffset + utf16Len(ln.text[:byteIdx2]),
		})
		segments = append(segments, segment)
		joinedTextOffset += utf16Len(segment)
		endOffset = ln.endOffset
		i++
	}

	return Blockquote{
		Text:       strings.Join(segments, "\n"),
		Sp:         spanFromOffsets(source, first.startOffset, endOffset),
		SegmentMap: segmentMap,
	}, i, true
}

func consumeParagraph(source string, lines []rawLine, startIdx int, plugins []ScannerPlugin) (Block, int) {
	first := lines[startIdx]

	endIdx := startIdx
	for endIdx+1 < len(lines) {
		candidateIdx := endIdx + 1
		candidate := lines[candidateIdx]
		if strings.TrimSpace(candidate.text) == "" {
			break
		}
		if headingStart.MatchString(candidate.text) {
			break
		}
		if ulRe.MatchString(candidate.text) {
			break
		}
		if olRe.MatchString(candidate.text) {
			break
		}
		if bqRe.MatchString(candidate.text) {
			break
		}
		if fenceRe.MatchString(candidate.text) {
			break
		}
		if rowRe.MatchString(candidate.text) {
			break
		}
		if isThematicBreak(candidate.text) {
			break
		}
		if _, _, ok := runPlugins(source, lines, candidateIdx, plugins); ok {
			break
		}
		endIdx++
	}

	last := lines[endIdx]
	startOffset := first.startOffset
	endOffset := last.endOffset
	return Paragraph{
		Text:       utf16Slice(source, startOffset, endOffset),
		Sp:         spanFromOffsets(source, startOffset, endOffset),
		SegmentMap: []SegmentOffset{{TextOffset: 0, SourceOffset: startOffset}},
	}, endIdx + 1
}

func tryFence(source string, lines []rawLine, startIdx int) (Block, int, bool) {
	if startIdx >= len(lines) {
		return nil, 0, false
	}
	start := lines[startIdx]
	openM := fenceRe.FindStringSubmatch(start.text)
	if openM == nil {
		return nil, 0, false
	}
	fenceMarker := openM[1]
	info := strings.TrimSpace(openM[2])

	i := startIdx + 1
	bodyStart := -1
	bodyEnd := -1
	endOffset := start.endOffset

	for i < len(lines) {
		ln := lines[i]
		closeM := fenceRe.FindStringSubmatch(ln.text)
		if closeM != nil && len(closeM[1]) >= len(fenceMarker) {
			endOffset = ln.endOffset
			break
		}
		if bodyStart == -1 {
			bodyStart = ln.startOffset
		}
		bodyEnd = ln.endOffset + 1 // +1 to include the '\n' after this line
		i++
	}

	body := ""
	if bodyStart != -1 && bodyEnd != -1 {
		body = utf16Slice(source, bodyStart, bodyEnd)
	}

	fallback := start.endOffset
	bs, be := fallback, fallback
	if bodyStart != -1 {
		bs = bodyStart
	}
	if bodyEnd != -1 {
		be = bodyEnd
	}
	return Fence{
		Info:     info,
		Body:     body,
		BodySpan: spanFromOffsets(source, bs, be),
		Sp:       spanFromOffsets(source, start.startOffset, endOffset),
	}, i + 1, true
}

func tryTable(source string, lines []rawLine, startIdx int) (Block, int, bool) {
	if startIdx+1 >= len(lines) {
		return nil, 0, false
	}
	headerLine := lines[startIdx]
	delimLine := lines[startIdx+1]

	if !rowRe.MatchString(headerLine.text) {
		return nil, 0, false
	}
	if !delimRe.MatchString(delimLine.text) {
		return nil, 0, false
	}

	headerCells, headerCellSpans := parseRowCells(headerLine.text, headerLine.startOffset, source)
	header := Row{
		Cells:     headerCells,
		CellSpans: headerCellSpans,
		Sp:        spanFromOffsets(source, headerLine.startOffset, headerLine.endOffset),
	}

	rows := make([]Row, 0)
	i := startIdx + 2
	for i < len(lines) {
		ln := lines[i]
		if !rowRe.MatchString(ln.text) {
			break
		}
		cells, cellSpans := parseRowCells(ln.text, ln.startOffset, source)
		rows = append(rows, Row{
			Cells:     cells,
			CellSpans: cellSpans,
			Sp:        spanFromOffsets(source, ln.startOffset, ln.endOffset),
		})
		i++
	}

	endOffset := delimLine.endOffset
	if len(rows) > 0 {
		endOffset = rows[len(rows)-1].Sp.EndOffset
	}
	return Table{
		Sp:     spanFromOffsets(source, headerLine.startOffset, endOffset),
		Header: header,
		Rows:   rows,
	}, i, true
}
