// Package bundle14 mirrors squares.steps.ts.
package bundle14

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	r.Stimulus("I warm up my mental math", func(s state) state { return s })

	// Two {int} slots; the handler takes the first capture and returns both
	// slot values positionally.
	r.Sensor("The square of {int} is {int}.", func(_ state, n int) []any {
		return []any{n, n * n}
	})

	return r.Done()
}
