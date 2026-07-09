// Package bundle02 mirrors counter.steps.ts.
package bundle02

import (
	"fmt"

	"github.com/oselvar/var/go/oselvar"
)

type state struct {
	count int
}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	r.Stimulus("I increment", func(s state) state {
		return state{count: s.count + 1}
	})

	r.Sensor("The count is {int}", func(s state, n int) error {
		if s.count != n {
			return fmt.Errorf("expected %d but got %d", n, s.count)
		}
		return nil
	})

	return r.Done()
}
