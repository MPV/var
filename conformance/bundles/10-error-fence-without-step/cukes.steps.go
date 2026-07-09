// Package bundle10 mirrors cukes.steps.ts.
package bundle10

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// The prose matches no step, so the `error` fence has nothing to run →
	// error-fence-without-step diagnostic, and the example is dropped.
	r.Stimulus("I have {int} cukes", func(s state, _ int) state { return s })

	return r.Done()
}
