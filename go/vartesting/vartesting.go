// Package vartesting is the Go stdlib `testing` adapter for Vár (ADR 0007). An
// author runs their specs with one ordinary test:
//
//	func TestVar(t *testing.T) { vartesting.Run(t, steps.Bundle()) }
//
// Run discovers .md specs via var.config.json, plans each against the bundle's
// registry, reconciles drift, and reports one t.Run sub-test per Markdown
// example — independently selectable via `go test -run`, with failures rendered
// anchored to the .md span.
package vartesting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oselvar/var/go/oselvar"
	"github.com/oselvar/var/go/varconfig"
	"github.com/oselvar/var/go/varcore"
	"github.com/oselvar/var/go/varrunner"
)

// ExampleOutcome is one example's result: a nil Err means it passed.
type ExampleOutcome struct {
	Name string
	Err  error
}

// SpecOutcome is one spec file's results.
type SpecOutcome struct {
	Path        string
	Source      string
	Diagnostics []varcore.Diagnostic
	Drifts      []varcore.Drift
	Examples    []ExampleOutcome
}

// Run drives the adapter from the current working directory. See RunAt.
func Run(t *testing.T, bundle *oselvar.Bundle) {
	t.Helper()
	RunAt(t, ".", bundle)
}

// RunAt drives the adapter from root, reporting one sub-test per example. Drift
// acknowledgment is via VAR_UPDATE (any non-empty value).
func RunAt(t *testing.T, root string, bundle *oselvar.Bundle) {
	t.Helper()
	update := os.Getenv("VAR_UPDATE") != ""
	specs, err := CollectOutcomes(root, bundle, update)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		spec := spec
		t.Run(spec.Path, func(t *testing.T) {
			for _, d := range spec.Diagnostics {
				d := d
				t.Run("diagnostic:"+string(d.Code), func(t *testing.T) {
					t.Errorf("%s: %s", spec.Path, d.Message)
				})
			}
			for _, dr := range spec.Drifts {
				dr := dr
				t.Run("drift", func(t *testing.T) {
					t.Errorf("%s:%d drift: %q — fix the step or accept it (VAR_UPDATE=1)", spec.Path, dr.Line, dr.Name)
				})
			}
			for _, ex := range spec.Examples {
				ex := ex
				t.Run(ex.Name, func(t *testing.T) {
					if ex.Err != nil {
						t.Errorf("%s", varrunner.RenderFailure(ex.Err, spec.Source, spec.Path))
					}
				})
			}
		})
	}
}

// CollectOutcomes runs every discovered spec and returns structured results
// (the testing-free core of the adapter, so it can be unit-tested). It performs
// the drift reconciliation side effect (writing var.lock.json on a clean run).
func CollectOutcomes(root string, bundle *oselvar.Bundle, update bool) ([]SpecOutcome, error) {
	cfg, err := varconfig.ReadVarConfig(root)
	if err != nil {
		return nil, err
	}
	specPaths, err := varrunner.FindSpecs(cfg.DocsInclude, cfg.DocsExclude, root)
	if err != nil {
		return nil, err
	}
	store := varrunner.NewFileBaselineStore(root)

	out := make([]SpecOutcome, 0, len(specPaths))
	for _, specPath := range specPaths {
		rel, err := filepath.Rel(root, specPath)
		if err != nil {
			rel = specPath
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(specPath)
		if err != nil {
			return nil, err
		}
		source := string(raw)
		doc, plan, err := varrunner.PlanSpec(rel, source, bundle.Registry)
		if err != nil {
			return nil, err
		}
		drifts := varcore.ReconcileDrift(store, rel, source, doc, plan, update)
		examples := make([]ExampleOutcome, 0)
		for _, q := range varcore.CollectExamples(plan, bundle.CreateContext) {
			examples = append(examples, ExampleOutcome{Name: q.Name, Err: q.Run()})
		}
		out = append(out, SpecOutcome{
			Path:        rel,
			Source:      source,
			Diagnostics: plan.Diagnostics,
			Drifts:      drifts,
			Examples:    examples,
		})
	}
	return out, nil
}
