// Package oselvar is the author-facing facade of the Go port of Vár. Authors
// define step definitions here; the runner/adapter execute them against Markdown
// specs. Port of the var facade (python var/, ts @oselvar/var).
//
// The package is named oselvar, not var, because var is a Go keyword.
//
// Registration uses an injected Registrar (ADR 0006) — no global mutable
// accumulator. State evolves by full replacement: a stimulus returns the whole
// next state value (ADR 0006), fitting Go's static typing.
package oselvar

import (
	"runtime"

	"github.com/oselvar/var/go/varcore"
)

// Bundle is what a step-definition fixture returns: the populated registry plus
// a constructor for a fresh initial state. The runner/conformance harness reads
// both.
type Bundle struct {
	Registry      varcore.Registry
	CreateContext func(name string) any
}

// Registrar accumulates step definitions for one state type S. Obtain one with
// DefineState, register with Stimulus/Sensor/DefineParameterType, then call
// Done.
type Registrar[S any] struct {
	reg varcore.Registry
}

// DefineState begins a set of step definitions over state type S.
func DefineState[S any]() *Registrar[S] {
	return &Registrar[S]{reg: varcore.CreateRegistry()}
}

// Stimulus registers a step that drives the software. fn must be
// func(S, captures...) (S, error) — it returns the whole next state.
func (r *Registrar[S]) Stimulus(expression string, fn any) {
	r.add(expression, fn, varcore.Stimulus)
}

// Sensor registers a read-only assertion step. fn must be
// func(S, captures...) (T, error) — its return is compared against the document.
func (r *Registrar[S]) Sensor(expression string, fn any) {
	r.add(expression, fn, varcore.Sensor)
}

func (r *Registrar[S]) add(expression string, fn any, kind varcore.StepKind) {
	_, file, line, _ := runtime.Caller(2) // caller of Stimulus/Sensor = the fixture
	reg, err := varcore.AddStep(r.reg, expression, file, line, fn, kind)
	if err != nil {
		panic(err) // duplicate expression is a static authoring error
	}
	r.reg = reg
}

// DefineParameterType registers a custom parameter type. parse transforms the
// captured string(s) into the value passed to handlers (nil = identity); format
// renders a value back to the document's notation (nil = default).
func (r *Registrar[S]) DefineParameterType(name, regexp string, parse func(...string) any, format varcore.ParameterFormat) {
	r.reg = varcore.DefineParameterType(r.reg, varcore.CustomParamType{
		Name:           name,
		Regexp:         regexp,
		Parse:          parse,
		UseForSnippets: true,
	}, format)
}

// Done returns the Bundle for this state type. CreateContext yields the zero
// value of S as the initial state (full-replacement model).
func (r *Registrar[S]) Done() *Bundle {
	return &Bundle{
		Registry: r.reg,
		CreateContext: func(string) any {
			var s S
			return s
		},
	}
}
