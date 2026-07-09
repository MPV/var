package steps

import (
	"fmt"

	"github.com/oselvar/var/go/oselvar"
)

func registerHelloVar(r *oselvar.Registrar[State]) {
	r.Stimulus("I greet {string}", func(s State, name string) State {
		s.Greeting = fmt.Sprintf("Hello, %s!", name)
		return s
	})

	r.Sensor("the greeting should be {string}", func(s State, _ string) string {
		return s.Greeting
	})

	r.Stimulus("expression `{int}+{int}`", func(s State, a, b int) State {
		s.Result = a + b
		return s
	})

	r.Sensor("evaluate to `{int}`", func(s State, _ int) int {
		return s.Result
	})
}
