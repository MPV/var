package varcore

import "fmt"

// registry.go — the immutable step-definition registry. Port of registry.py /
// registry.ts.
//
// Kept free of the cucumber-expressions library: expression compilation and
// matching happen in the matcher (plan stage), which builds a cucumber
// ParameterTypeRegistry from CustomParamTypes on demand. The registry stage
// needs only the expression text (for parameterTypeNames) and the custom
// parameter-type declarations (for the artifact).

// StepHandler is a step's handler, stored opaquely; the executor invokes it via
// reflection at the trace stage.
type StepHandler any

// ParameterFormat renders a value as the document's notation (presentation
// only — never part of matching or comparison).
type ParameterFormat func(any) string

// CustomParamType is a user-defined parameter type. Regexp is the bare source
// string, which is what serializes into the registry artifact.
type CustomParamType struct {
	Name                 string
	Regexp               string
	Parse                func(...string) any
	UseForSnippets       bool
	PreferForRegexpMatch bool
}

// StepRegistration is one registered step definition.
type StepRegistration struct {
	Expression           string
	ExpressionSourceFile string
	ExpressionSourceLine int
	Handler              StepHandler
	Kind                 StepKind
}

// Registry is an immutable set of step definitions plus custom parameter types.
// AddStep/DefineParameterType return a new Registry.
type Registry struct {
	Steps            []StepRegistration
	CustomParamTypes []CustomParamType
	Formats          map[string]ParameterFormat
}

// CreateRegistry returns an empty registry.
func CreateRegistry() Registry {
	return Registry{Formats: map[string]ParameterFormat{}}
}

// AddStep appends a compiled-later step to registry, returning a new Registry.
// Errors on a duplicate expression, mirroring registry.ts / registry.py.
func AddStep(r Registry, expression, sourceFile string, sourceLine int, handler StepHandler, kind StepKind) (Registry, error) {
	for _, s := range r.Steps {
		if s.Expression == expression {
			return r, fmt.Errorf("duplicate step definition for %q at %s:%d and %s:%d",
				expression, s.ExpressionSourceFile, s.ExpressionSourceLine, sourceFile, sourceLine)
		}
	}
	steps := make([]StepRegistration, len(r.Steps), len(r.Steps)+1)
	copy(steps, r.Steps)
	steps = append(steps, StepRegistration{
		Expression:           expression,
		ExpressionSourceFile: sourceFile,
		ExpressionSourceLine: sourceLine,
		Handler:              handler,
		Kind:                 kind,
	})
	return Registry{Steps: steps, CustomParamTypes: r.CustomParamTypes, Formats: r.Formats}, nil
}

// DefineParameterType registers a custom parameter type, returning a new
// Registry. Port of define_parameter_type.
func DefineParameterType(r Registry, pt CustomParamType, format ParameterFormat) Registry {
	types := make([]CustomParamType, len(r.CustomParamTypes), len(r.CustomParamTypes)+1)
	copy(types, r.CustomParamTypes)
	types = append(types, pt)

	formats := r.Formats
	if format != nil {
		formats = make(map[string]ParameterFormat, len(r.Formats)+1)
		for k, v := range r.Formats {
			formats[k] = v
		}
		formats[pt.Name] = format
	}
	return Registry{Steps: r.Steps, CustomParamTypes: types, Formats: formats}
}
