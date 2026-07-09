package gotesting_test

import (
	"testing"

	"github.com/oselvar/var-examples/go-testing/steps"
	"github.com/oselvar/var/go/vartesting"
)

// TestVar runs every Markdown spec in this project as tests: one t.Run sub-test
// per example. Run with `go test`; select one with, e.g.,
// `go test -run 'TestVar/hello-var.md'`.
func TestVar(t *testing.T) {
	vartesting.Run(t, steps.Bundle())
}
