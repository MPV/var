package steps

import (
	"strconv"
	"strings"

	"github.com/oselvar/var/go/oselvar"
)

// toRoman converts a positive integer to its Roman-numeral form.
func toRoman(n int) string {
	values := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	symbols := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range values {
		for n >= v {
			b.WriteString(symbols[i])
			n -= v
		}
	}
	return b.String()
}

func registerRomanNumerals(r *oselvar.Registrar[State]) {
	// Header-bound table: the step returns the computed columns for each row.
	r.Sensor("a decimal and a roman number", func(_ State, row map[string]string) map[string]string {
		decimal, _ := strconv.Atoi(row["decimal"])
		return map[string]string{"decimal": row["decimal"], "roman": toRoman(decimal)}
	})
}
