package varcore

import (
	"strings"
	"unicode"
)

// sentences.go — split a block of plain text into sentences on . ! ? and \n,
// skipping terminators inside backtick spans and double-quoted strings and
// treating common abbreviations as non-breaking. Port of sentences.py /
// sentences.ts.
//
// Offsets are UTF-16 code-unit offsets into the original text. The algorithm
// works over runes (matching the reference's per-code-point logic), tracking a
// rune-index→UTF-16 map for the emitted offsets.

type sentence struct {
	text        string
	startOffset int
	endOffset   int
}

var abbreviations = map[string]bool{
	"e.g.": true, "i.e.": true, "etc.": true, "cf.": true, "vs.": true,
}

// buildRuneToU16 returns a slice where m[i] is the UTF-16 offset of runes[i]
// (and m[len] is the total UTF-16 length).
func buildRuneToU16(runes []rune) []int {
	m := make([]int, len(runes)+1)
	u16 := 0
	for i, r := range runes {
		m[i] = u16
		if r > 0xFFFF {
			u16 += 2
		} else {
			u16++
		}
	}
	m[len(runes)] = u16
	return m
}

func isInsideNumberOrAbbrev(runes []rune, dotPos int) bool {
	var prev, next rune
	if dotPos > 0 {
		prev = runes[dotPos-1]
	}
	if dotPos+1 < len(runes) {
		next = runes[dotPos+1]
	}
	if unicode.IsDigit(prev) && unicode.IsDigit(next) {
		return true
	}
	for abbrev := range abbreviations {
		al := len([]rune(abbrev))
		start := dotPos + 1 - al
		if start < 0 {
			start = 0
		}
		if string(runes[start:dotPos+1]) == abbrev {
			return true
		}
	}
	if unicode.IsLower(next) {
		return true
	}
	return false
}

func pushSegment(out *[]sentence, runes []rune, startRune, endRune int, runeToU16 []int) {
	if endRune <= startRune {
		return
	}
	raw := string(runes[startRune:endRune])
	stripped := strings.TrimSpace(raw)
	if stripped == "" {
		return
	}
	// Leading/trailing whitespace are all ASCII (1 rune each here).
	lead := len([]rune(raw)) - len([]rune(strings.TrimLeftFunc(raw, unicode.IsSpace)))
	trail := len([]rune(raw)) - len([]rune(strings.TrimRightFunc(raw, unicode.IsSpace)))
	trimmedStart := startRune + lead
	trimmedEnd := endRune - trail
	*out = append(*out, sentence{
		text:        stripped,
		startOffset: runeToU16[trimmedStart],
		endOffset:   runeToU16[trimmedEnd],
	})
}

// splitSentences splits text into sentences with UTF-16 offsets. Port of
// split_sentences / splitSentences.
func splitSentences(text string) []sentence {
	runes := []rune(text)
	n := len(runes)
	runeToU16 := buildRuneToU16(runes)
	out := make([]sentence, 0)

	// Mark no-split zones (backtick spans, double-quoted strings).
	skip := make([]bool, n)
	j := 0
	for j < n {
		c := runes[j]
		if c == '`' {
			close := indexRune(runes, '`', j+1)
			if close == -1 {
				break
			}
			for k := j; k <= close; k++ {
				skip[k] = true
			}
			j = close + 1
			continue
		}
		if c == '"' {
			close := indexRune(runes, '"', j+1)
			if close == -1 {
				break
			}
			for k := j; k <= close; k++ {
				skip[k] = true
			}
			j = close + 1
			continue
		}
		j++
	}

	i := 0
	segmentStart := 0
	for i < n {
		if skip[i] {
			i++
			continue
		}
		ch := runes[i]
		if ch == '\n' || ch == '.' || ch == '!' || ch == '?' {
			if ch == '.' && isInsideNumberOrAbbrev(runes, i) {
				i++
				continue
			}
			end := i + 1
			pushSegment(&out, runes, segmentStart, end, runeToU16)
			i = end
			for i < n && (runes[i] == ' ' || runes[i] == '\n') {
				i++
			}
			segmentStart = i
			continue
		}
		i++
	}

	pushSegment(&out, runes, segmentStart, n, runeToU16)
	return out
}

func indexRune(runes []rune, target rune, from int) int {
	for k := from; k < len(runes); k++ {
		if runes[k] == target {
			return k
		}
	}
	return -1
}
