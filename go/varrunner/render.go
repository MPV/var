package varrunner

import (
	"fmt"
	"strings"

	"github.com/oselvar/var/go/varcore"
)

// RenderFailure renders a step failure as a human-readable, markdown-anchored
// string, dispatching on the concrete error type. Port of render_failure. It
// reuses the core's diff payloads (never re-deriving failure text).
func RenderFailure(err error, source, path string) string {
	if c, ok := varcore.AsCellMismatch(err); ok {
		lines := []string{fmt.Sprintf("Cell mismatch in %s:", path)}
		any := false
		for _, cell := range c.Cells {
			if cell.OK {
				continue
			}
			any = true
			lines = append(lines, fmt.Sprintf("  line %d | column '%s' — expected: %q, actual: %q",
				cell.Span.StartLine, cell.Column, cell.Expected, cell.Actual))
		}
		if !any {
			lines = append(lines, "  (no failing cells)")
		}
		return strings.Join(lines, "\n")
	}
	if d, ok := varcore.AsDocStringMismatch(err); ok {
		return fmt.Sprintf("Doc string mismatch at line %d:\n  expected: %q\n  actual:   %q",
			d.Diff.Span.StartLine, d.Diff.Expected, d.Diff.Actual)
	}
	if rs, ok := varcore.AsReturnShape(err); ok {
		return rs.Message
	}
	return fmt.Sprintf("%T: %s", err, err.Error())
}
