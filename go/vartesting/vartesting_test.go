package vartesting_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/oselvar/var/go/oselvar"
	"github.com/oselvar/var/go/varcore"
	"github.com/oselvar/var/go/vartesting"
)

type counterState struct{ count int }

func counterBundle() *oselvar.Bundle {
	r := oselvar.DefineState[counterState]()
	r.Stimulus("I increment", func(s counterState) counterState {
		return counterState{count: s.count + 1}
	})
	r.Sensor("The count is {int}", func(s counterState, n int) error {
		if s.count != n {
			return fmt.Errorf("expected %d but got %d", n, s.count)
		}
		return nil
	})
	return r.Done()
}

func writeSpec(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "var.config.json"), []byte(`{"docs":{"include":["*.md"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterPassingExample(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "# Counter\n\nI increment. The count is 1.\n")
	outcomes, err := vartesting.CollectOutcomes(dir, counterBundle(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || len(outcomes[0].Examples) != 1 {
		t.Fatalf("expected 1 spec / 1 example, got %+v", outcomes)
	}
	if outcomes[0].Examples[0].Err != nil {
		t.Errorf("expected pass, got %v", outcomes[0].Examples[0].Err)
	}
	// A clean run writes the baseline.
	if _, err := os.Stat(filepath.Join(dir, "var.lock.json")); err != nil {
		t.Errorf("expected var.lock.json to be written: %v", err)
	}
}

func TestAdapterFailingExample(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "# Counter\n\nI increment. The count is 2.\n")
	outcomes, err := vartesting.CollectOutcomes(dir, counterBundle(), false)
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[0].Examples[0].Err == nil {
		t.Error("expected the example to fail (count 1 != 2)")
	}
}

func TestAdapterSurfacesDrift(t *testing.T) {
	dir := t.TempDir()
	writeSpec(t, dir, "# Counter\n\nI increment. The count is 1.\n")
	source, _ := os.ReadFile(filepath.Join(dir, "spec.md"))
	// A baseline that recorded this paragraph as an example...
	lock := varcore.VarLock{Version: 1, Specs: map[string]varcore.SpecBaseline{
		"spec.md": {
			SourceHash: varcore.HashSource(string(source)),
			Examples:   []varcore.BaselineExample{{Name: "I increment. The count is 1", Line: 3}},
		},
	}}
	if err := os.WriteFile(filepath.Join(dir, "var.lock.json"), []byte(varcore.StringifyVarLock(lock)), 0o644); err != nil {
		t.Fatal(err)
	}
	// ...but now run with an empty registry, so nothing matches → drift.
	empty := oselvar.DefineState[counterState]().Done()
	outcomes, err := vartesting.CollectOutcomes(dir, empty, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes[0].Drifts) == 0 {
		t.Error("expected drift to be surfaced when a recorded example no longer matches")
	}
}
