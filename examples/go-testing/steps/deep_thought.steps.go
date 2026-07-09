package steps

import "github.com/oselvar/var/go/oselvar"

func registerDeepThought(r *oselvar.Registrar[State]) {
	r.Sensor("life, the universe and everything is {int}", func(_ State, _ int) int {
		return 42
	})
}
