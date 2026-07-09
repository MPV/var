// Package bundle06 mirrors echo.steps.ts.
package bundle06

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// Returns the WRONG string (bare); the core compares it to the doc string
	// and reports a doc-string-mismatch.
	r.Sensor("I echo the following:", func(_ state, _ string) string {
		return "goodbye"
	})

	return r.Done()
}
