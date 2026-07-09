// Package steps holds the Go step definitions for the go-testing sample. A
// single unified State type is used across specs; the executor keeps a fresh
// State per step file (per-file state), so stateless specs simply ignore it.
package steps

import "github.com/oselvar/var/go/oselvar"

// State is the shared context for every spec in this project.
type State struct {
	Greeting string
	Result   int
}

// Bundle assembles all specs' step definitions into one registry. With the
// injected-Registrar model, each spec contributes via a register function.
func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[State]()
	registerHelloVar(r)
	registerDeepThought(r)
	registerRomanNumerals(r)
	registerTablesAndDocstrings(r)
	registerYahtzee(r)
	return r.Done()
}
