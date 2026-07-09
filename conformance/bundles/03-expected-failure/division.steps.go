// Package bundle03 mirrors division.steps.ts.
package bundle03

import (
	"errors"

	"github.com/oselvar/var/go/oselvar"
)

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	r.Stimulus("I divide {int} by {int}", func(s state, a, b int) (state, error) {
		if b == 0 {
			return s, errors.New("division by zero")
		}
		return s, nil
	})

	return r.Done()
}
