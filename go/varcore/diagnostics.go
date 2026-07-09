package varcore

import (
	"fmt"
	"strings"
)

// diagnostics.go — planner diagnostics. Port of diagnostics.py / diagnostics.ts
// (the subset the planner and drift need).

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type DiagnosticCode string

const (
	CodeAmbiguousMatch        DiagnosticCode = "ambiguous-match"
	CodeErrorFenceWithoutStep DiagnosticCode = "error-fence-without-step"
	CodeDrift                 DiagnosticCode = "drift"
)

type Candidate struct {
	Expression string
	SourceFile string
	SourceLine int
}

type Diagnostic struct {
	Code     DiagnosticCode
	Severity Severity
	Message  string
	Span     Span
}

type AmbiguousInput struct {
	Text       string
	Span       Span
	Candidates []Candidate
}

func ambiguousMatch(in AmbiguousInput) Diagnostic {
	lines := make([]string, len(in.Candidates))
	for i, c := range in.Candidates {
		lines[i] = fmt.Sprintf("  '%s'    at %s:%d", c.Expression, c.SourceFile, c.SourceLine)
	}
	return Diagnostic{
		Severity: SeverityError,
		Code:     CodeAmbiguousMatch,
		Message:  fmt.Sprintf("Ambiguous step: \"%s\"\nMatched by:\n%s", in.Text, strings.Join(lines, "\n")),
		Span:     in.Span,
	}
}

func driftDetected(name string, span Span) Diagnostic {
	return Diagnostic{
		Severity: SeverityError,
		Code:     CodeDrift,
		Message: fmt.Sprintf(
			"This paragraph was an example and no longer matches any step (drift): \"%s\".\n"+
				"Fix the step so it matches again, or accept it as prose (run in update mode).", name),
		Span: span,
	}
}

func errorFenceWithoutStep(span Span) Diagnostic {
	return Diagnostic{
		Severity: SeverityError,
		Code:     CodeErrorFenceWithoutStep,
		Message: "This `error` fence marks the example as expected-to-fail, " +
			"but the example has no step to run.",
		Span: span,
	}
}
