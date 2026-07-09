// Package bundle08 mirrors greet.steps.ts.
package bundle08

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	r.Stimulus("I greet {string}", func(s state, _ string) state { return s })

	return r.Done()
}
