// Package bundle04 mirrors echo.steps.ts.
package bundle04

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// The doc string is this sensor's only slot, so it is returned bare; the
	// core compares it against the input; equal content passes.
	r.Sensor("I echo the following:", func(_ state, doc string) string {
		return doc
	})

	return r.Done()
}
