// Package bundle05 mirrors cukes.steps.ts.
package bundle05

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// Both expressions match "I have 5 cukes" → ambiguous-match diagnostic.
	r.Stimulus("I have {int} cukes", func(s state, _ int) state { return s })
	r.Stimulus("I have 5 cukes", func(s state) state { return s })

	return r.Done()
}
