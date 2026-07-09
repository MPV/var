package varcore

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// plan.go — produce an ExecutionPlan from a VarDoc + Registry. Port of plan.py /
// plan.ts. Matches step expressions against each text-bearing block, attaches
// trailing tables / fences, detects header-bound tables, and collects
// diagnostics.

// DocString is a fenced code block attached to a step.
type DocString struct {
	Content     string
	ContentType string
	Span        Span
}

// PlannedStep is one matched step with its resolved source spans and attachments.
type PlannedStep struct {
	Text       string
	MatchSpan  Span
	ParamSpans []Span
	StepDef    StepRegistration
	Args       []any
	Formats    []ParameterFormat
	DataTable  *Table
	DocString  *DocString
}

// HeaderBinding describes the paragraph shared by all rows of a header-bound table.
type HeaderBinding struct {
	MatchSpan  Span
	ParamSpans []Span
	StepDef    StepRegistration
}

// RowCheck is one header-bound column check (column name, expected cell value,
// source span). Lives here (rather than cell_diff) until the trace stage.
type RowCheck struct {
	Column string
	Value  string
	Span   Span
}

// PlannedExample is one example ready to execute.
type PlannedExample struct {
	Name                 string
	ScopeStack           []string
	Span                 Span
	Steps                []PlannedStep
	HeaderBinding        *HeaderBinding
	RowChecks            []RowCheck
	ExpectedOutcome      string  // "" (pass) or "fail"
	ExpectedErrorMessage *string // set only when a non-empty error message is expected
}

// ExecutionPlan is the whole planned document.
type ExecutionPlan struct {
	VarDoc      VarDoc
	Examples    []PlannedExample
	Diagnostics []Diagnostic
}

var wsRe = regexp.MustCompile(`\s+`)

func textBearing(b Block) (text string, segMap []SegmentOffset, ok bool) {
	switch v := b.(type) {
	case Paragraph:
		return v.Text, v.SegmentMap, true
	case ListItem:
		return v.Text, v.SegmentMap, true
	case Blockquote:
		return v.Text, v.SegmentMap, true
	}
	return "", nil, false
}

func liftSegmentOffset(segMap []SegmentOffset, textOffset int) int {
	if len(segMap) == 0 {
		panic("empty segment_map")
	}
	best := segMap[0]
	for _, entry := range segMap {
		if entry.TextOffset <= textOffset {
			best = entry
		}
	}
	return best.SourceOffset + (textOffset - best.TextOffset)
}

func liftSpan(source string, b Block, start, end int) Span {
	_, segMap, ok := textBearing(b)
	if !ok {
		return b.blockSpan()
	}
	startSrc := liftSegmentOffset(segMap, start)
	endSrc := liftSegmentOffset(segMap, end)
	return spanFromOffsets(source, startSrc, endSrc)
}

type blockPlan struct {
	steps       []hit
	ambiguities []ambiguityCollision
}

func planBlock(text string, cr *compiledRegistry) blockPlan {
	allSteps := make([]hit, 0)
	allAmbiguities := make([]ambiguityCollision, 0)

	for _, sen := range splitSentences(text) {
		hits := findHits(sen.text, cr)
		adjusted := make([]hit, len(hits))
		for i, h := range hits {
			ps := make([]paramSpanRel, len(h.paramSpans))
			for k, p := range h.paramSpans {
				ps[k] = paramSpanRel{start: p.start + sen.startOffset, end: p.end + sen.startOffset}
			}
			adjusted[i] = hit{
				expression: h.expression,
				stepDef:    h.stepDef,
				matchStart: h.matchStart + sen.startOffset,
				matchEnd:   h.matchEnd + sen.startOffset,
				args:       h.args,
				paramSpans: ps,
				formats:    h.formats,
			}
		}
		resolved := resolveHits(adjusted)
		if resolved.ambiguous {
			allAmbiguities = append(allAmbiguities, resolved.collisions...)
		} else if len(resolved.steps) > 0 {
			allSteps = append(allSteps, resolved.steps...)
		}
	}
	return blockPlan{steps: allSteps, ambiguities: allAmbiguities}
}

func isWordChar(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// wordOffset returns the UTF-16 offset of word as a whole word in haystack, or
// -1. Hand-ports plan.ts's wordOffset (`(?<![^\W_])word(?![^\W_])`); RE2 has no
// lookaround, so boundaries are checked directly. A word char is a Unicode
// letter or digit (underscore is a boundary).
func wordOffset(haystack, word string) int {
	from := 0
	for {
		idx := strings.Index(haystack[from:], word)
		if idx < 0 {
			return -1
		}
		byteStart := from + idx
		byteEnd := byteStart + len(word)
		okBefore := true
		if byteStart > 0 {
			r, _ := utf8.DecodeLastRuneInString(haystack[:byteStart])
			if isWordChar(r) {
				okBefore = false
			}
		}
		okAfter := true
		if byteEnd < len(haystack) {
			r, _ := utf8.DecodeRuneInString(haystack[byteEnd:])
			if isWordChar(r) {
				okAfter = false
			}
		}
		if okBefore && okAfter {
			return utf16OfByte(haystack, byteStart)
		}
		from = byteStart + 1
	}
}

type headerBound struct {
	table       Table
	lastStep    PlannedStep
	headerSpans []Span
}

func detectHeaderBound(source string, ex Example, stepsByBlock map[int][]PlannedStep) *headerBound {
	body := ex.Body
	for idx := 1; idx < len(body); idx++ {
		here := body[idx]
		table, isTable := here.(Table)
		if !isTable {
			continue
		}
		above := body[idx-1]
		aboveText, _, ok := textBearing(above)
		if !ok {
			continue
		}
		steps := stepsByBlock[idx-1]
		if len(steps) == 0 {
			continue
		}
		headerCells := table.Header.Cells
		offsets := make([]int, len(headerCells))
		allFound := true
		for i, cell := range headerCells {
			offsets[i] = wordOffset(aboveText, cell)
			if offsets[i] < 0 {
				allFound = false
			}
		}
		if !allFound {
			continue
		}
		headerSpans := make([]Span, len(headerCells))
		for i := range headerCells {
			headerSpans[i] = liftSpan(source, above, offsets[i], offsets[i]+utf16Len(headerCells[i]))
		}
		return &headerBound{table: table, lastStep: steps[len(steps)-1], headerSpans: headerSpans}
	}
	return nil
}

// deriveExampleName mirrors derive_example_name.
func deriveExampleName(body []Block) string {
	var primary string
	found := false
	for _, b := range body {
		if t, _, ok := textBearing(b); ok {
			primary = t
			found = true
			break
		}
	}
	if !found {
		return ""
	}
	name := strings.TrimSpace(wsRe.ReplaceAllString(primary, " "))
	return stripTrailingTerminator(name)
}

// stripTrailingTerminator removes a single trailing . ! or ? (mirrors
// /[.!?]$/ replace).
func stripTrailingTerminator(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeLastRuneInString(s)
	if r == '.' || r == '!' || r == '?' {
		return s[:len(s)-size]
	}
	return s
}

// BuildPlan mirrors plan(). It produces an ExecutionPlan from var_doc + registry.
func BuildPlan(varDoc VarDoc, registry Registry) (ExecutionPlan, error) {
	cr, err := compileRegistry(registry)
	if err != nil {
		return ExecutionPlan{}, err
	}

	examples := make([]PlannedExample, 0)
	diagnostics := make([]Diagnostic, 0)

	for _, ex := range varDoc.Examples {
		hadAmbiguous := false
		stepsByBlock := map[int][]PlannedStep{}

		for idx, block := range ex.Body {
			blockText, _, ok := textBearing(block)
			if !ok {
				continue
			}
			result := planBlock(blockText, cr)

			for _, collision := range result.ambiguities {
				span := liftSpan(varDoc.Source, block, collision.matchStart, collision.matchEnd)
				candidates := make([]Candidate, len(collision.candidates))
				for i, c := range collision.candidates {
					candidates[i] = Candidate{
						Expression: c.expression,
						SourceFile: c.stepDef.ExpressionSourceFile,
						SourceLine: c.stepDef.ExpressionSourceLine,
					}
				}
				diagnostics = append(diagnostics, ambiguousMatch(AmbiguousInput{
					Text:       utf16Slice(blockText, collision.matchStart, collision.matchEnd),
					Span:       span,
					Candidates: candidates,
				}))
				hadAmbiguous = true
			}

			if !hadAmbiguous && len(result.steps) > 0 {
				blockSteps := make([]PlannedStep, len(result.steps))
				for i, h := range result.steps {
					paramSpans := make([]Span, len(h.paramSpans))
					for k, p := range h.paramSpans {
						paramSpans[k] = liftSpan(varDoc.Source, block, p.start, p.end)
					}
					blockSteps[i] = PlannedStep{
						Text:       utf16Slice(blockText, h.matchStart, h.matchEnd),
						MatchSpan:  liftSpan(varDoc.Source, block, h.matchStart, h.matchEnd),
						ParamSpans: paramSpans,
						StepDef:    h.stepDef,
						Args:       h.args,
						Formats:    h.formats,
					}
				}
				stepsByBlock[idx] = blockSteps
			}
		}

		// Header-bound table detection.
		var bound *headerBound
		if !hadAmbiguous {
			bound = detectHeaderBound(varDoc.Source, ex, stepsByBlock)
		}

		if bound != nil {
			headerBinding := &HeaderBinding{
				MatchSpan:  bound.lastStep.MatchSpan,
				ParamSpans: bound.headerSpans,
				StepDef:    bound.lastStep.StepDef,
			}
			for _, row := range bound.table.Rows {
				rowObject := map[string]string{}
				for i, cellName := range bound.table.Header.Cells {
					if i < len(row.Cells) {
						rowObject[cellName] = row.Cells[i]
					} else {
						rowObject[cellName] = ""
					}
				}
				rowStep := PlannedStep{
					Text:       bound.lastStep.Text,
					MatchSpan:  row.Sp,
					ParamSpans: bound.lastStep.ParamSpans,
					StepDef:    bound.lastStep.StepDef,
					Args:       append(append([]any{}, bound.lastStep.Args...), rowObject),
					Formats:    bound.lastStep.Formats,
				}
				rowChecks := make([]RowCheck, len(bound.table.Header.Cells))
				for i, cellName := range bound.table.Header.Cells {
					value := ""
					span := row.Sp
					if i < len(row.Cells) {
						value = row.Cells[i]
					}
					if i < len(row.CellSpans) {
						span = row.CellSpans[i]
					}
					rowChecks[i] = RowCheck{Column: cellName, Value: value, Span: span}
				}
				examples = append(examples, PlannedExample{
					Name:          strings.Join(row.Cells, " / "),
					ScopeStack:    append(append([]string{}, ex.ScopeStack...), bound.lastStep.Text),
					Span:          row.Sp,
					Steps:         []PlannedStep{rowStep},
					HeaderBinding: headerBinding,
					RowChecks:     rowChecks,
				})
			}
			continue
		}

		// Error fence detection.
		var errorFence *Fence
		for i := range ex.Body {
			if f, ok := ex.Body[i].(Fence); ok && f.Info == "error" {
				fc := f
				errorFence = &fc
				break
			}
		}

		// Attach trailing table / fence to the last step in a block.
		type attachment struct {
			dataTable *Table
			docString *DocString
		}
		attachments := map[int]attachment{}
		for idx := 1; idx < len(ex.Body); idx++ {
			here := ex.Body[idx]
			if tbl, ok := here.(Table); ok {
				if _, has := stepsByBlock[idx-1]; has {
					prev := attachments[idx-1]
					t := tbl
					attachments[idx-1] = attachment{dataTable: &t, docString: prev.docString}
				}
			} else if fnc, ok := here.(Fence); ok && fnc.Info != "error" {
				if _, has := stepsByBlock[idx-1]; has {
					prev := attachments[idx-1]
					attachments[idx-1] = attachment{
						dataTable: prev.dataTable,
						docString: &DocString{Content: fnc.Body, ContentType: fnc.Info, Span: fnc.BodySpan},
					}
				}
			}
		}

		// Rebuild the final step list, applying attachments.
		finalSteps := make([]PlannedStep, 0)
		for idx := 0; idx < len(ex.Body); idx++ {
			blockSteps := stepsByBlock[idx]
			attach, hasAttach := attachments[idx]
			for sIdx, step := range blockSteps {
				if sIdx == len(blockSteps)-1 && hasAttach {
					step.DataTable = attach.dataTable
					step.DocString = attach.docString
				}
				finalSteps = append(finalSteps, step)
			}
		}

		var runnableSteps []PlannedStep
		if !hadAmbiguous {
			runnableSteps = finalSteps
		}

		if errorFence != nil && len(runnableSteps) == 0 {
			diagnostics = append(diagnostics, errorFenceWithoutStep(errorFence.Sp))
		}

		if len(finalSteps) == 0 && !hadAmbiguous {
			continue
		}

		expectedOutcome := ""
		var expectedErrorMessage *string
		if errorFence != nil {
			expectedOutcome = "fail"
			msg := strings.TrimSpace(errorFence.Body)
			if msg != "" {
				expectedErrorMessage = &msg
			}
		}

		examples = append(examples, PlannedExample{
			Name:                 deriveExampleName(ex.Body),
			ScopeStack:           ex.ScopeStack,
			Span:                 ex.Sp,
			Steps:                runnableSteps,
			ExpectedOutcome:      expectedOutcome,
			ExpectedErrorMessage: expectedErrorMessage,
		})
	}

	return ExecutionPlan{VarDoc: varDoc, Examples: examples, Diagnostics: diagnostics}, nil
}
