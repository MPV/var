// Package bundle07 mirrors report.steps.ts.
package bundle07

import "github.com/oselvar/var/go/oselvar"

type state struct{}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// Header-bound row step: returns its computed columns; the core diffs them
	// against the row cells. score 99 ≠ 10 → cell-mismatch.
	r.Sensor("I report the score and grade", func(_ state) map[string]string {
		return map[string]string{"score": "99", "grade": "A"}
	})

	return r.Done()
}
