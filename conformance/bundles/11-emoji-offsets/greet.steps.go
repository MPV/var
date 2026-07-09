// Package bundle11 mirrors greet.steps.ts.
package bundle11

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	r.Sensor("I greet {string}", func(_ state, _ string) any { return nil })

	return r.Done()
}
