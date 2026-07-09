// Package bundle09 mirrors boom.steps.ts.
package bundle09

import (
	"errors"

	"github.com/oselvar/var/go/oselvar"
)

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// Throws a message that does NOT contain the expected substring, so the
	// expected-failure is not satisfied → the example fails.
	r.Stimulus("I always boom", func(s state) (state, error) {
		return s, errors.New("actual different error")
	})

	return r.Done()
}
