package varcore

// conformance.go — projections of pipeline output to the canonical wire shapes
// compared against conformance/bundles/*/golden/*.json. Port of conformance.py /
// conformance.ts. Artifacts are built from map[string]any / []any so
// CanonicalStringify inherits encoding/json's recursive map-key sort.
//
// Staged like the reference: ToVarDocArtifact (parse) first; ToRegistryArtifact,
// ToPlanArtifact, and RunConformance (trace) are added as their stages land.

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
