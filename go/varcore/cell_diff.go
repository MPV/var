package varcore

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// cell_diff.go — compare row/table step returns against the authored Markdown
// cells. Port of cell_diff.py / cell-diff.ts.

// CellDiff is the verdict for one checked column.
type CellDiff struct {
	Column        string
	Span          Span
	Expected      string
	Actual        string
	OK            bool
	ExpectedValue any
	ActualValue   any
	Formatted     bool
}

// renderCellValue applies display rules 2-4 of the mismatch-rendering chain
// (rule 1, the parameter type's format, is param_diff's job). A string renders
// as-is, other primitives via their default string form, anything else via a
// Go-native fallback (deliberately outside conformance — pin object actuals with
// a format).
func renderCellValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "None"
	case string:
		return v
	case bool:
		if v {
			return "True"
		}
		return "False"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// CellMismatchError is raised when returned columns don't all match.
type CellMismatchError struct {
	Cells []CellDiff
}

func (e *CellMismatchError) Error() string {
	parts := make([]string, len(e.Cells))
	for i, c := range e.Cells {
		parts[i] = fmt.Sprintf("%s: expected %s but was %s", c.Column, c.Expected, c.Actual)
	}
	return strings.Join(parts, "; ")
}

func isCellMismatchError(e error) (*CellMismatchError, bool) {
	var c *CellMismatchError
	if e != nil && errors.As(e, &c) {
		return c, true
	}
	return nil, false
}

// AsCellMismatch reports whether err is a CellMismatchError (for adapters).
func AsCellMismatch(err error) (*CellMismatchError, bool) { return isCellMismatchError(err) }

// ReturnShapeError signals the step returned the wrong type or shape — an author
// mistake, not a value diff.
type ReturnShapeError struct{ Message string }

func (e *ReturnShapeError) Error() string { return e.Message }

// AsReturnShape reports whether err is a ReturnShapeError (for adapters).
func AsReturnShape(err error) (*ReturnShapeError, bool) {
	var r *ReturnShapeError
	if err != nil && errors.As(err, &r) {
		return r, true
	}
	return nil, false
}

// compareRow compares a row step's returned map against the row's cells. Only
// columns present on returned are checked. A non-map (or nil) return checks
// nothing. Port of compare_row.
func compareRow(returned any, checks []RowCheck) []CellDiff {
	if returned == nil {
		return nil
	}
	rv := reflect.ValueOf(returned)
	if rv.Kind() != reflect.Map {
		return nil
	}
	diffs := make([]CellDiff, 0)
	for _, check := range checks {
		mv := rv.MapIndex(reflect.ValueOf(check.Column))
		if !mv.IsValid() {
			continue
		}
		actual := renderCellValue(mv.Interface())
		diffs = append(diffs, CellDiff{
			Column:   check.Column,
			Span:     check.Span,
			Expected: check.Value,
			Actual:   actual,
			OK:       actual == check.Value,
		})
	}
	return diffs
}

// compareTable compares a whole-table step's returned rows against the input
// table (array-of-arrays or array-of-records). Port of compare_table. Not
// exercised by the current corpus but kept for parity.
func compareTable(returned any, table Table) ([]CellDiff, error) {
	if returned == nil {
		return nil, nil
	}
	rv := reflect.ValueOf(returned)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, &ReturnShapeError{Message: fmt.Sprintf("expected a table (array of rows), got %T", returned)}
	}
	columns := table.Header.Cells
	dataRows := table.Rows
	if rv.Len() != len(dataRows) {
		return nil, &ReturnShapeError{Message: fmt.Sprintf("expected %d row(s), got %d", len(dataRows), rv.Len())}
	}

	rows := make([]any, rv.Len())
	for i := range rows {
		rows[i] = rv.Index(i).Interface()
	}
	allArrays := true
	allRecords := true
	for _, r := range rows {
		k := reflect.ValueOf(r).Kind()
		if k != reflect.Slice && k != reflect.Array {
			allArrays = false
		}
		if k != reflect.Map {
			allRecords = false
		}
	}
	if !allArrays && !allRecords {
		return nil, &ReturnShapeError{Message: "table rows must be all arrays or all objects"}
	}

	diffs := make([]CellDiff, 0)
	for i, row := range dataRows {
		ret := reflect.ValueOf(rows[i])
		if allArrays && ret.Len() != len(columns) {
			return nil, &ReturnShapeError{Message: fmt.Sprintf("row %d: expected %d column(s), got %d", i, len(columns), ret.Len())}
		}
		for j, column := range columns {
			var actualValue any
			if allArrays {
				actualValue = ret.Index(j).Interface()
			} else {
				mv := ret.MapIndex(reflect.ValueOf(column))
				if !mv.IsValid() {
					return nil, &ReturnShapeError{Message: fmt.Sprintf("row %d: missing column %q", i, column)}
				}
				actualValue = mv.Interface()
			}
			expected := ""
			if j < len(row.Cells) {
				expected = row.Cells[j]
			}
			actual := renderCellValue(actualValue)
			span := row.Sp
			if j < len(row.CellSpans) {
				span = row.CellSpans[j]
			}
			diffs = append(diffs, CellDiff{Column: column, Span: span, Expected: expected, Actual: actual, OK: actual == expected})
		}
	}
	return diffs, nil
}
