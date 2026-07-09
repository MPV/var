package steps

import (
	"fmt"
	"strings"

	"github.com/oselvar/var/go/oselvar"
)

func registerTablesAndDocstrings(r *oselvar.Registrar[State]) {
	// Whole-table mode: the table arrives as rows (header row first). It is this
	// sensor's only slot, so return the reproduced table bare — Vár compares
	// every cell.
	r.Sensor("Uppercase each one:", func(_ State, rows [][]string) []map[string]string {
		out := make([]map[string]string, 0, len(rows)-1)
		for _, row := range rows[1:] {
			before := row[0]
			out = append(out, map[string]string{"before": before, "after": strings.ToUpper(before)})
		}
		return out
	})

	// Doc-string mode: two slots ({word} plus the trailing doc string), so
	// return one element per slot.
	r.Sensor("Greet {word}:", func(_ State, name string, _ string) []any {
		return []any{name, fmt.Sprintf("Hello, %s!\n", name)}
	})
}
