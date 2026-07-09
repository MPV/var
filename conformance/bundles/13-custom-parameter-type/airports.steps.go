// Package bundle13 mirrors airports.steps.ts.
package bundle13

import (
	"fmt"
	"strings"

	"github.com/oselvar/var/go/oselvar"
)

type state struct {
	dest string
}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// Custom {airport} parameter type: IATA code, lowercased by the parse
	// function. The lowercasing is asserted by the sensor, so an identity parse
	// would fail this bundle — proving parse functions execute.
	r.DefineParameterType("airport", `[A-Z]{3}`, func(args ...string) any {
		return strings.ToLower(args[0])
	}, nil)

	r.Stimulus("I fly to {airport}", func(_ state, dest string) state {
		return state{dest: dest}
	})

	r.Sensor("The destination code is {word}", func(s state, expected string) error {
		cleaned := strings.TrimRight(expected, ".!?")
		if s.dest != cleaned {
			return fmt.Errorf("expected %s but got %s", cleaned, s.dest)
		}
		return nil
	})

	return r.Done()
}
