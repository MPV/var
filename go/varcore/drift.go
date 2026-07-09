package varcore

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
)

// drift.go — spec drift detection. Port of drift.py / drift.ts. A paragraph the
// committed var.lock.json baseline recorded as an example that now matches no
// step. Byte-identical to the other ports so var.lock.json is shared.

// DriftSimilarityThreshold: a baseline example is re-identified by an exact name
// match, else the most word-similar paragraph at or above this threshold. Ported
// byte-identically.
const DriftSimilarityThreshold = 0.5

// BaselineExample is one example-producing paragraph as recorded in the baseline.
type BaselineExample struct {
	Name string
	Line int
}

// SpecBaseline is the committed baseline for one spec file.
type SpecBaseline struct {
	SourceHash string
	Examples   []BaselineExample
}

// VarLock is the whole var.lock.json: every spec keyed by its POSIX path.
type VarLock struct {
	Version int
	Specs   map[string]SpecBaseline
}

// Drift is a paragraph the baseline says was an example and now matches no step.
type Drift struct {
	Name string
	Line int
	Span Span
}

// BaselineStore is the persistence port for var.lock.json. The core owns the
// format; adapters move only raw text.
type BaselineStore interface {
	Read() (string, bool)
	Write(contents string)
}

func within(inner, outer Span) bool {
	return inner.StartOffset >= outer.StartOffset && inner.EndOffset <= outer.EndOffset
}

func isLive(candidateSpan Span, plan ExecutionPlan) bool {
	for _, pe := range plan.Examples {
		if within(pe.Span, candidateSpan) {
			return true
		}
	}
	return false
}

var tokenRe = regexp.MustCompile(`[^\W_]+`)

func tokenize(text string) map[string]bool {
	out := map[string]bool{}
	for _, tok := range tokenRe.FindAllString(strings.ToLower(text), -1) {
		out[tok] = true
	}
	return out
}

func similarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// LiveExamples returns the current example-producing paragraphs, in document
// order.
func LiveExamples(varDoc VarDoc, plan ExecutionPlan) []BaselineExample {
	out := make([]BaselineExample, 0)
	for _, candidate := range varDoc.Examples {
		if isLive(candidate.Sp, plan) {
			out = append(out, BaselineExample{
				Name: deriveExampleName(candidate.Body),
				Line: candidate.Sp.StartLine,
			})
		}
	}
	return out
}

// DeriveSpecBaseline builds the full baseline record for a spec.
func DeriveSpecBaseline(source string, varDoc VarDoc, plan ExecutionPlan) SpecBaseline {
	return SpecBaseline{SourceHash: HashSource(source), Examples: LiveExamples(varDoc, plan)}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// DetectDrift returns paragraphs the baseline recorded as examples that now
// match zero steps. Port of detect_drift. baseline == nil → no drift.
func DetectDrift(baseline *SpecBaseline, varDoc VarDoc, plan ExecutionPlan) []Drift {
	if baseline == nil {
		return nil
	}
	candidates := varDoc.Examples
	tokens := make([]map[string]bool, len(candidates))
	live := make([]bool, len(candidates))
	for i, c := range candidates {
		tokens[i] = tokenize(deriveExampleName(c.Body))
		live[i] = isLive(c.Sp, plan)
	}

	drifts := make([]Drift, 0)
	for _, b := range baseline.Examples {
		bTokens := tokenize(b.Name)
		bestIdx := -1
		bestScore := 0.0
		for i, candidate := range candidates {
			score := similarity(bTokens, tokens[i])
			if score < DriftSimilarityThreshold {
				continue
			}
			line := candidate.Sp.StartLine
			bestLine := 0
			if bestIdx >= 0 {
				bestLine = candidates[bestIdx].Sp.StartLine
			}
			if bestIdx < 0 || score > bestScore ||
				(score == bestScore && abs(line-b.Line) < abs(bestLine-b.Line)) {
				bestIdx = i
				bestScore = score
			}
		}
		if bestIdx < 0 || live[bestIdx] {
			continue
		}
		candidate := candidates[bestIdx]
		drifts = append(drifts, Drift{Name: b.Name, Line: candidate.Sp.StartLine, Span: candidate.Sp})
	}
	return drifts
}

// DriftDiagnostics projects drifts onto the shared Diagnostic rail.
func DriftDiagnostics(drifts []Drift) []Diagnostic {
	out := make([]Diagnostic, len(drifts))
	for i, d := range drifts {
		out[i] = driftDetected(d.Name, d.Span)
	}
	return out
}

// ReconcileDrift reconciles one spec's baseline against a BaselineStore. update
// accepts all drift (re-record, report nothing); otherwise detect drift and
// rewrite the baseline only on a clean run. Port of reconcile_drift.
func ReconcileDrift(store BaselineStore, specPath, source string, varDoc VarDoc, plan ExecutionPlan, update bool) []Drift {
	var lock *VarLock
	if text, ok := store.Read(); ok && text != "" {
		lock = ParseVarLock(text)
	}
	var baseline *SpecBaseline
	if lock != nil {
		if b, ok := lock.Specs[specPath]; ok {
			baseline = &b
		}
	}
	var drifts []Drift
	if !update {
		drifts = DetectDrift(baseline, varDoc, plan)
	}
	if update || len(drifts) == 0 {
		nextSpec := DeriveSpecBaseline(source, varDoc, plan)
		specs := map[string]SpecBaseline{}
		if lock != nil {
			for k, v := range lock.Specs {
				specs[k] = v
			}
		}
		specs[specPath] = nextSpec
		store.Write(StringifyVarLock(VarLock{Version: 1, Specs: specs}))
	}
	return drifts
}

// --- serialization (var.lock.json uses its own format, NOT canonical_json) ---

type baselineExampleJSON struct {
	Name string `json:"name"`
	Line int    `json:"line"`
}

type specBaselineJSON struct {
	SourceHash string                `json:"sourceHash"`
	Examples   []baselineExampleJSON `json:"examples"`
}

type varLockJSON struct {
	Version int                         `json:"version"`
	Specs   map[string]specBaselineJSON `json:"specs"`
}

// StringifyVarLock serializes var.lock.json deterministically: spec paths sorted
// (map keys sort in encoding/json), examples in document order, insertion-order
// keys otherwise (struct field order), 2-space indent, non-ASCII raw, trailing
// newline. Port of stringify_var_lock.
func StringifyVarLock(lock VarLock) string {
	specs := make(map[string]specBaselineJSON, len(lock.Specs))
	for path, sb := range lock.Specs {
		examples := make([]baselineExampleJSON, 0, len(sb.Examples))
		for _, e := range sb.Examples {
			examples = append(examples, baselineExampleJSON{Name: e.Name, Line: e.Line})
		}
		specs[path] = specBaselineJSON{SourceHash: sb.SourceHash, Examples: examples}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(varLockJSON{Version: 1, Specs: specs})
	return buf.String()
}

// ParseVarLock parses var.lock.json; nil on malformed input (treated as no
// baseline). Port of parse_var_lock.
func ParseVarLock(text string) *VarLock {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil
	}
	if v, ok := parsed["version"].(float64); !ok || v != 1 {
		return nil
	}
	specsRaw, ok := parsed["specs"].(map[string]any)
	if !ok {
		return nil
	}
	specs := map[string]SpecBaseline{}
	for path, value := range specsRaw {
		baseline := parseSpecBaseline(value)
		if baseline == nil {
			return nil
		}
		specs[path] = *baseline
	}
	return &VarLock{Version: 1, Specs: specs}
}

func parseSpecBaseline(value any) *SpecBaseline {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	sourceHash, ok := m["sourceHash"].(string)
	if !ok {
		return nil
	}
	examplesRaw, ok := m["examples"].([]any)
	if !ok {
		return nil
	}
	examples := make([]BaselineExample, 0, len(examplesRaw))
	for _, item := range examplesRaw {
		e := parseBaselineExample(item)
		if e == nil {
			return nil
		}
		examples = append(examples, *e)
	}
	return &SpecBaseline{SourceHash: sourceHash, Examples: examples}
}

func parseBaselineExample(value any) *BaselineExample {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	name, ok := m["name"].(string)
	if !ok {
		return nil
	}
	// JSON numbers decode to float64; a valid line is integral.
	lineF, ok := m["line"].(float64)
	if !ok || lineF != float64(int(lineF)) {
		return nil
	}
	return &BaselineExample{Name: name, Line: int(lineF)}
}
