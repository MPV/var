package varcore

import (
	"errors"
	"fmt"
)

// doc_string_diff.go — compare a doc-string step's return against the fence body.
// Port of doc_string_diff.py / doc-string-diff.ts.

// DocStringDiff is a doc-string content difference.
type DocStringDiff struct {
	Span     Span
	Expected string
	Actual   string
}

// compareDocString compares a doc-string step's return against the fence body.
// nil → no check; equal → nil; unequal → a diff; non-string → ReturnShapeError.
func compareDocString(returned any, content string, span Span) (*DocStringDiff, error) {
	if returned == nil {
		return nil, nil
	}
	s, ok := returned.(string)
	if !ok {
		return nil, &ReturnShapeError{Message: fmt.Sprintf("expected a doc string (string), got %T", returned)}
	}
	if s == content {
		return nil, nil
	}
	return &DocStringDiff{Span: span, Expected: content, Actual: s}, nil
}

// DocStringMismatchError is raised when a doc-string return differs from the
// authored content.
type DocStringMismatchError struct {
	Diff DocStringDiff
}

func (e *DocStringMismatchError) Error() string {
	return fmt.Sprintf("doc string: expected %q but was %q", e.Diff.Expected, e.Diff.Actual)
}

// AsDocStringMismatch reports whether err is a DocStringMismatchError (for adapters).
func AsDocStringMismatch(err error) (*DocStringMismatchError, bool) {
	return isDocStringMismatchError(err)
}

func isDocStringMismatchError(e error) (*DocStringMismatchError, bool) {
	var d *DocStringMismatchError
	if e != nil && errors.As(e, &d) {
		return d, true
	}
	return nil, false
}
