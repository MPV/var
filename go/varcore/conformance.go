package varcore

// conformance.go — projections of pipeline output to the canonical wire shapes
// compared against conformance/bundles/*/golden/*.json. Port of conformance.py /
// conformance.ts. Artifacts are built from map[string]any / []any so
// CanonicalStringify inherits encoding/json's recursive map-key sort.

import (
	"path/filepath"
	"strings"
)

// fileStem returns the file stem: "path/to/foo.steps.go" -> "foo.steps". Port of
// _file_stem / fileStem — strips the final extension so a step-def file
// serializes identically across every language's fixture.
func fileStem(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// toFailureArtifact projects an execution error to a FailureArtifact dict. Port
// of to_failure_artifact / toFailureArtifact. line and anchor are deterministic
// source positions (never scraped from a stack), so every port reproduces them.
func toFailureArtifact(err error, matchSpan Span) map[string]any {
	line := matchSpan.StartLine
	anchor := spanMap(failureAnchor(err, matchSpan))
	if c, ok := isCellMismatchError(err); ok {
		cells := make([]any, 0)
		for _, cell := range c.Cells {
			if !cell.OK {
				cells = append(cells, map[string]any{
					"column":   cell.Column,
					"expected": cell.Expected,
					"actual":   cell.Actual,
					"span":     spanMap(cell.Span),
				})
			}
		}
		return map[string]any{"kind": "cell-mismatch", "line": line, "anchor": anchor, "cells": cells}
	}
	if d, ok := isDocStringMismatchError(err); ok {
		return map[string]any{
			"kind": "doc-string-mismatch", "line": line, "anchor": anchor,
			"diff": map[string]any{
				"expected": d.Diff.Expected,
				"actual":   d.Diff.Actual,
				"span":     spanMap(d.Diff.Span),
			},
		}
	}
	if _, ok := err.(*ReturnShapeError); ok {
		return map[string]any{"kind": "return-shape", "line": line, "anchor": anchor}
	}
	if isUnexpectedPassError(err) {
		return map[string]any{"kind": "unexpected-pass", "line": line, "anchor": anchor}
	}
	return map[string]any{"kind": "thrown", "line": line, "anchor": anchor}
}

// RunConformance runs all examples and returns the trace artifact. Port of
// run_conformance / runConformance — the trace is built inline from recorded
// step observations (there is no separate toTraceArtifact).
func RunConformance(varDoc VarDoc, registry Registry, createContext func(string) any) (map[string]any, error) {
	plan, err := BuildPlan(varDoc, registry)
	if err != nil {
		return nil, err
	}

	traceExamples := make([]any, 0)
	for _, ex := range plan.Examples {
		obs := make([]StepObservation, 0)
		runErr := executeExample(plan, ex, createContext, func(o StepObservation) {
			obs = append(obs, o)
		})
		outcome := "pass"
		if runErr != nil {
			outcome = "fail"
		}

		steps := make([]any, len(ex.Steps))
		for i, step := range ex.Steps {
			ordinal := i + 1
			var chosen *StepObservation
			var last *StepObservation
			for j := range obs {
				if obs[j].Ordinal == ordinal {
					last = &obs[j]
					if obs[j].Outcome == "fail" {
						chosen = &obs[j]
						break
					}
				}
			}
			if chosen == nil {
				chosen = last
			}
			stepOutcome := "skipped"
			if chosen != nil {
				stepOutcome = chosen.Outcome
			}
			stepDict := map[string]any{
				"exampleName":       ex.Name,
				"ordinal":           ordinal,
				"stepText":          step.Text,
				"matchedExpression": step.StepDef.Expression,
				"contextKey": map[string]any{
					"exampleName": ex.Name,
					"stepFile":    fileStem(step.StepDef.ExpressionSourceFile),
				},
				"outcome": stepOutcome,
			}
			if stepOutcome == "fail" {
				var e error
				if chosen != nil {
					e = chosen.Err
				}
				stepDict["failure"] = toFailureArtifact(e, step.MatchSpan)
			}
			steps[i] = stepDict
		}

		traceExamples = append(traceExamples, map[string]any{
			"name":    ex.Name,
			"outcome": outcome,
			"steps":   steps,
		})
	}

	return map[string]any{"examples": traceExamples}, nil
}

// ToRegistryArtifact projects a Registry to the wire dict for the registry
// artifact. Port of to_registry_artifact / toRegistryArtifact. parameterTypeNames
// come from the expression text (see parameterTypeNames); custom parameter types
// serialize as {name, regexp}.
func ToRegistryArtifact(r Registry) map[string]any {
	steps := make([]any, len(r.Steps))
	for i, s := range r.Steps {
		names := parameterTypeNames(s.Expression)
		nameList := make([]any, len(names))
		for j, n := range names {
			nameList[j] = n
		}
		steps[i] = map[string]any{
			"expression":         s.Expression,
			"parameterTypeNames": nameList,
		}
	}
	paramTypes := make([]any, len(r.CustomParamTypes))
	for i, p := range r.CustomParamTypes {
		paramTypes[i] = map[string]any{
			"name":   p.Name,
			"regexp": p.Regexp,
		}
	}
	return map[string]any{
		"steps":          steps,
		"parameterTypes": paramTypes,
	}
}

// ToPlanArtifact projects an ExecutionPlan to the wire dict for the plan
// artifact. Port of to_plan_artifact / toPlanArtifact. A step's args come from
// its param spans (value = the source slice, parameterType = the name in source
// order), not from the matched values.
func ToPlanArtifact(plan ExecutionPlan) map[string]any {
	source := plan.VarDoc.Source

	stepMap := func(step PlannedStep) map[string]any {
		names := parameterTypeNames(step.StepDef.Expression)
		paramSpans := make([]any, len(step.ParamSpans))
		args := make([]any, len(step.ParamSpans))
		for i, sp := range step.ParamSpans {
			paramSpans[i] = spanMap(sp)
			var pt any
			if i < len(names) {
				pt = names[i]
			}
			args[i] = map[string]any{
				"value":         utf16Slice(source, sp.StartOffset, sp.EndOffset),
				"parameterType": pt,
			}
		}
		m := map[string]any{
			"text":              step.Text,
			"matchSpan":         spanMap(step.MatchSpan),
			"paramSpans":        paramSpans,
			"matchedExpression": step.StepDef.Expression,
			"args":              args,
		}
		if step.DataTable != nil {
			m["dataTable"] = blockMap(*step.DataTable)
		}
		if step.DocString != nil {
			m["docString"] = map[string]any{
				"content":     step.DocString.Content,
				"contentType": step.DocString.ContentType,
				"span":        spanMap(step.DocString.Span),
			}
		}
		return m
	}

	examples := make([]any, len(plan.Examples))
	for i, ex := range plan.Examples {
		scope := make([]any, len(ex.ScopeStack))
		for j, s := range ex.ScopeStack {
			scope[j] = s
		}
		steps := make([]any, len(ex.Steps))
		for j, s := range ex.Steps {
			steps[j] = stepMap(s)
		}
		outcome := ex.ExpectedOutcome
		if outcome == "" {
			outcome = "pass"
		}
		m := map[string]any{
			"name":            ex.Name,
			"scopeStack":      scope,
			"span":            spanMap(ex.Span),
			"expectedOutcome": outcome,
			"steps":           steps,
		}
		if ex.ExpectedErrorMessage != nil {
			m["expectedErrorMessage"] = *ex.ExpectedErrorMessage
		}
		examples[i] = m
	}

	diagnostics := make([]any, len(plan.Diagnostics))
	for i, d := range plan.Diagnostics {
		diagnostics[i] = map[string]any{
			"code":     string(d.Code),
			"severity": string(d.Severity),
			"span":     spanMap(d.Span),
		}
	}

	return map[string]any{
		"examples":    examples,
		"diagnostics": diagnostics,
	}
}

func spanMap(s Span) map[string]any {
	return map[string]any{
		"startOffset": s.StartOffset,
		"endOffset":   s.EndOffset,
		"startLine":   s.StartLine,
		"startCol":    s.StartCol,
		"endLine":     s.EndLine,
		"endCol":      s.EndCol,
	}
}

func segmentMap(so SegmentOffset) map[string]any {
	return map[string]any{
		"textOffset":   so.TextOffset,
		"sourceOffset": so.SourceOffset,
	}
}

func segmentsMap(segs []SegmentOffset) []any {
	out := make([]any, len(segs))
	for i, so := range segs {
		out[i] = segmentMap(so)
	}
	return out
}

func rowMap(r Row) map[string]any {
	spans := make([]any, len(r.CellSpans))
	for i, cs := range r.CellSpans {
		spans[i] = spanMap(cs)
	}
	cells := make([]any, len(r.Cells))
	for i, c := range r.Cells {
		cells[i] = c
	}
	return map[string]any{
		"cells":     cells,
		"cellSpans": spans,
		"span":      spanMap(r.Sp),
	}
}

// blockMap projects one block to its wire dict, dispatching on concrete type
// (the Go analogue of the reference's isinstance chain).
func blockMap(b Block) map[string]any {
	switch v := b.(type) {
	case Paragraph:
		return map[string]any{
			"kind":       v.kind(),
			"text":       v.Text,
			"span":       spanMap(v.Sp),
			"segmentMap": segmentsMap(v.SegmentMap),
		}
	case Heading:
		return map[string]any{
			"kind":  v.kind(),
			"level": v.Level,
			"text":  v.Text,
			"span":  spanMap(v.Sp),
		}
	case ListItem:
		return map[string]any{
			"kind":       v.kind(),
			"text":       v.Text,
			"span":       spanMap(v.Sp),
			"segmentMap": segmentsMap(v.SegmentMap),
			"ordered":    v.Ordered,
			"markerSpan": spanMap(v.MarkerSpan),
		}
	case Blockquote:
		return map[string]any{
			"kind":       v.kind(),
			"text":       v.Text,
			"span":       spanMap(v.Sp),
			"segmentMap": segmentsMap(v.SegmentMap),
		}
	case Table:
		rows := make([]any, len(v.Rows))
		for i, r := range v.Rows {
			rows[i] = rowMap(r)
		}
		return map[string]any{
			"kind":   v.kind(),
			"span":   spanMap(v.Sp),
			"header": rowMap(v.Header),
			"rows":   rows,
		}
	case Fence:
		return map[string]any{
			"kind":     v.kind(),
			"span":     spanMap(v.Sp),
			"info":     v.Info,
			"body":     v.Body,
			"bodySpan": spanMap(v.BodySpan),
		}
	case ThematicBreak:
		return map[string]any{
			"kind": v.kind(),
			"span": spanMap(v.Sp),
		}
	}
	panic("unknown block type")
}

func exampleMap(ex Example) map[string]any {
	body := make([]any, len(ex.Body))
	for i, b := range ex.Body {
		body[i] = blockMap(b)
	}
	scope := make([]any, len(ex.ScopeStack))
	for i, s := range ex.ScopeStack {
		scope[i] = s
	}
	return map[string]any{
		"scopeStack": scope,
		"span":       spanMap(ex.Sp),
		"body":       body,
	}
}

// ToVarDocArtifact projects a VarDoc to the wire dict for the var-doc artifact.
// Port of to_var_doc_artifact / toVarDocArtifact.
func ToVarDocArtifact(doc VarDoc) map[string]any {
	examples := make([]any, len(doc.Examples))
	for i, ex := range doc.Examples {
		examples[i] = exampleMap(ex)
	}
	orphans := make([]any, len(doc.OrphanAttachments))
	for i, b := range doc.OrphanAttachments {
		orphans[i] = blockMap(b)
	}
	return map[string]any{
		"path":              doc.Path,
		"examples":          examples,
		"orphanAttachments": orphans,
	}
}
