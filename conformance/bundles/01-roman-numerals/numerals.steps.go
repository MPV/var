// Package bundle01 is the Go step-definition fixture for the roman-numerals
// bundle. Each conformance bundle declares its own package (bundleNN) because
// the shared bundle directory names (01-roman-numerals) are not valid Go
// package identifiers. Mirrors numerals.steps.ts.
package bundle01

import (
	"fmt"
	"strings"

	"github.com/oselvar/var/go/oselvar"
)

type state struct {
	result string
}

var roman = map[int]string{1: "I", 4: "IV", 9: "IX", 40: "XL"}

// Bundle returns the populated step registry and initial-state constructor.
func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	r.Stimulus("I convert {int} to roman numerals", func(_ state, n int) state {
		return state{result: roman[n]}
	})

	r.Sensor("The result is {word}", func(s state, expected string) error {
		// {word} greedily captures trailing sentence punctuation; strip one.
		cleaned := strings.TrimRight(expected, ".!?")
		if s.result != cleaned {
			return fmt.Errorf("expected %s but got %s", cleaned, s.result)
		}
		return nil
	})

	return r.Done()
}
