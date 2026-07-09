package varcore

import (
	"regexp"
	"strings"

	cukeexpr "github.com/cucumber/cucumber-expressions/go/v20"
)

// matcher.go — match step expressions against sentence text. Port of matcher.py /
// matcher.ts. Uses the cucumber-expressions Go library for regex compilation and
// matching; ports only var's own hit-resolution and offset-shifting around it.
//
// The library reports match offsets as byte offsets (Go regexp); every offset is
// converted to a UTF-16 code unit before leaving this module.

type paramSpanRel struct {
	start int // UTF-16 offset within the sentence
	end   int
}

type hit struct {
	expression string
	stepDef    StepRegistration
	matchStart int // UTF-16 offset in sentence
	matchEnd   int
	args       []any
	paramSpans []paramSpanRel
	formats    []ParameterFormat
}

type ambiguityCollision struct {
	matchStart int
	matchEnd   int
	candidates []hit
}

type resolvedSteps struct {
	ambiguous  bool
	steps      []hit
	collisions []ambiguityCollision
}

// compiledStep pairs a registration with its compiled expression and the
// un-anchored pattern used for substring scanning.
type compiledStep struct {
	step       StepRegistration
	expr       cukeexpr.Expression
	unanchored *regexp.Regexp
}

// compiledRegistry is a Registry with every expression compiled once.
type compiledRegistry struct {
	steps   []compiledStep
	formats map[string]ParameterFormat
}

// compileRegistry builds the cucumber ParameterTypeRegistry from the custom
// parameter types, then compiles every step expression. Port of the eager
// compilation registry.py does in add_step.
func compileRegistry(r Registry) (*compiledRegistry, error) {
	ptReg := cukeexpr.NewParameterTypeRegistry()
	for _, ct := range r.CustomParamTypes {
		re, err := regexp.Compile(ct.Regexp)
		if err != nil {
			return nil, err
		}
		var transform func(...*string) interface{}
		if ct.Parse != nil {
			parse := ct.Parse
			transform = func(gs ...*string) interface{} {
				ss := make([]string, len(gs))
				for i, g := range gs {
					if g != nil {
						ss[i] = *g
					}
				}
				return parse(ss...)
			}
		}
		pt, err := cukeexpr.NewParameterType(ct.Name, []*regexp.Regexp{re}, "", transform, ct.UseForSnippets, ct.PreferForRegexpMatch, false)
		if err != nil {
			return nil, err
		}
		if err := ptReg.DefineParameterType(pt); err != nil {
			return nil, err
		}
	}

	steps := make([]compiledStep, 0, len(r.Steps))
	for _, s := range r.Steps {
		expr, err := cukeexpr.NewCucumberExpression(s.Expression, ptReg)
		if err != nil {
			return nil, err
		}
		steps = append(steps, compiledStep{
			step:       s,
			expr:       expr,
			unanchored: unanchoredPattern(expr),
		})
	}
	return &compiledRegistry{steps: steps, formats: r.Formats}, nil
}

// unanchoredPattern strips the leading ^ and trailing $ from a compiled
// expression's regexp so it can find substring matches (mirrors
// cloneRegexpWithGlobal / _make_unanchored_pattern).
func unanchoredPattern(expr cukeexpr.Expression) *regexp.Regexp {
	source := expr.Regexp().String()
	source = strings.TrimPrefix(source, "^")
	source = strings.TrimSuffix(source, "$")
	return regexp.MustCompile(source)
}

// utf16OfByte returns the UTF-16 offset corresponding to a byte index in s.
func utf16OfByte(s string, byteIdx int) int {
	return utf16Len(s[:byteIdx])
}

// findHits returns every expression match found anywhere in sentence. Port of
// find_hits.
func findHits(sentence string, cr *compiledRegistry) []hit {
	hits := make([]hit, 0)
	for _, cs := range cr.steps {
		pos := 0
		for pos <= len(sentence) {
			loc := cs.unanchored.FindStringIndex(sentence[pos:])
			if loc == nil {
				break
			}
			matchByteStart := pos + loc[0]
			matchByteEnd := pos + loc[1]
			matchedText := sentence[matchByteStart:matchByteEnd]

			arguments, _ := cs.expr.Match(matchedText)

			args := make([]any, 0, len(arguments))
			formats := make([]ParameterFormat, 0, len(arguments))
			paramSpans := make([]paramSpanRel, 0, len(arguments))
			for _, arg := range arguments {
				args = append(args, arg.GetValue())
				formats = append(formats, cr.formats[arg.ParameterType().Name()])
				g := arg.Group()
				gs, ge := g.Start(), g.End()
				if gs >= 0 && ge >= 0 {
					absStart := matchByteStart + gs
					absEnd := matchByteStart + ge
					paramSpans = append(paramSpans, paramSpanRel{
						start: utf16OfByte(sentence, absStart),
						end:   utf16OfByte(sentence, absEnd),
					})
				}
			}

			hits = append(hits, hit{
				expression: cs.step.Expression,
				stepDef:    cs.step,
				matchStart: utf16OfByte(sentence, matchByteStart),
				matchEnd:   utf16OfByte(sentence, matchByteEnd),
				args:       args,
				paramSpans: paramSpans,
				formats:    formats,
			})

			if len(matchedText) == 0 {
				pos = matchByteStart + 1
			} else {
				pos = matchByteEnd
			}
		}
	}
	return hits
}

// resolveHits selects the best non-overlapping hits, or reports ambiguities.
// Port of resolve_hits.
func resolveHits(hits []hit) resolvedSteps {
	if len(hits) == 0 {
		return resolvedSteps{}
	}

	sorted := make([]hit, len(hits))
	copy(sorted, hits)
	// Sort by match_start ascending, then by length descending. Stable sort to
	// preserve registration order among equal keys (matches Python's sorted).
	stableSort(sorted, func(a, b hit) bool {
		if a.matchStart != b.matchStart {
			return a.matchStart < b.matchStart
		}
		return (a.matchEnd - a.matchStart) > (b.matchEnd - b.matchStart)
	})

	collisions := make([]ambiguityCollision, 0)
	i := 0
	for i < len(sorted) {
		here := sorted[i]
		hereLen := here.matchEnd - here.matchStart
		tied := []hit{here}
		j := i + 1
		for j < len(sorted) {
			cand := sorted[j]
			if cand.matchStart == here.matchStart && cand.matchEnd-cand.matchStart == hereLen {
				tied = append(tied, cand)
				j++
			} else {
				break
			}
		}
		if len(tied) > 1 {
			collisions = append(collisions, ambiguityCollision{
				matchStart: here.matchStart,
				matchEnd:   here.matchEnd,
				candidates: tied,
			})
		}
		i = j
	}

	if len(collisions) > 0 {
		return resolvedSteps{ambiguous: true, collisions: collisions}
	}

	steps := make([]hit, 0)
	cursor := -1
	for _, h := range sorted {
		if h.matchStart < cursor {
			continue
		}
		steps = append(steps, h)
		cursor = h.matchEnd
	}
	return resolvedSteps{steps: steps}
}

// stableSort is an insertion sort (stable) over hits by less. The hit counts per
// sentence are tiny, so an O(n^2) stable sort is simplest and matches Python's
// stable sorted().
func stableSort(s []hit, less func(a, b hit) bool) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && less(s[j], s[j-1]); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
