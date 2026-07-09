// Package bundle15 mirrors money.steps.ts.
package bundle15

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/oselvar/var/go/oselvar"
)

type state struct{}

type money struct {
	currency string
	value    float64
}

func Bundle() *oselvar.Bundle {
	r := oselvar.DefineState[state]()

	// Custom {money} parameter type with a format — the inverse of parse,
	// rendering a value back in the document's notation. The golden pins the
	// formatted actual ("£2.60"), proving every port renders parameter
	// mismatches through format identically.
	r.DefineParameterType("money", `£\d+\.\d{2}`,
		func(args ...string) any {
			v, _ := strconv.ParseFloat(strings.TrimPrefix(args[0], "£"), 64)
			return money{currency: "GBP", value: v}
		},
		func(m any) string {
			return fmt.Sprintf("£%.2f", m.(money).value)
		})

	// Returns the WRONG money on purpose.
	r.Sensor("The late fee is {money}", func(_ state, _ money) money {
		return money{currency: "GBP", value: 2.6}
	})

	return r.Done()
}
