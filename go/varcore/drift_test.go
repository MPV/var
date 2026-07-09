package varcore

import (
	"reflect"
	"testing"
)

// drift_test.go — translation of test_drift.py / drift.test.ts. Drift is
// unit-gated (no conformance golden).

func noop(...any) {}

func withdrawReg(t *testing.T, withStep bool) Registry {
	t.Helper()
	r := CreateRegistry()
	if withStep {
		var err error
		r, err = AddStep(r, "I withdraw {int}", "steps.go", 1, noop, Stimulus)
		if err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func romanReg(t *testing.T, withStep bool) Registry {
	t.Helper()
	r := CreateRegistry()
	if withStep {
		var err error
		r, err = AddStep(r, "a decimal and a roman number", "steps.go", 1, noop, Sensor)
		if err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func planFor(t *testing.T, doc VarDoc, r Registry) ExecutionPlan {
	t.Helper()
	p, err := BuildPlan(doc, r)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func bare(drifts []Drift) [][2]any {
	out := make([][2]any, len(drifts))
	for i, d := range drifts {
		out[i] = [2]any{d.Name, d.Line}
	}
	return out
}

type memoryStore struct {
	contents string
	set      bool
}

func (m *memoryStore) Read() (string, bool) { return m.contents, m.set }
func (m *memoryStore) Write(c string)       { m.contents, m.set = c, true }

func TestLiveExamplesRecordsOnePerExample(t *testing.T) {
	doc := Parse("w.md", "I withdraw 40.", nil)
	got := LiveExamples(doc, planFor(t, doc, withdrawReg(t, true)))
	want := []BaselineExample{{Name: "I withdraw 40", Line: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNeverMatchedParagraphNotLive(t *testing.T) {
	doc := Parse("w.md", "Just some prose.", nil)
	if got := LiveExamples(doc, planFor(t, doc, withdrawReg(t, true))); len(got) != 0 {
		t.Errorf("expected no live examples, got %v", got)
	}
}

func TestDeriveSpecBaselineCarriesHash(t *testing.T) {
	source := "I withdraw 40."
	doc := Parse("w.md", source, nil)
	b := DeriveSpecBaseline(source, doc, planFor(t, doc, withdrawReg(t, true)))
	if b.SourceHash != HashSource(source) {
		t.Errorf("hash mismatch")
	}
	if !reflect.DeepEqual(b.Examples, []BaselineExample{{Name: "I withdraw 40", Line: 1}}) {
		t.Errorf("examples = %v", b.Examples)
	}
}

func TestNoBaselineNoDrift(t *testing.T) {
	doc := Parse("w.md", "I withdraw 40.", nil)
	if got := DetectDrift(nil, doc, planFor(t, doc, withdrawReg(t, true))); len(got) != 0 {
		t.Errorf("expected no drift, got %v", got)
	}
}

func TestUnchangedNoDrift(t *testing.T) {
	source := "I withdraw 40."
	doc := Parse("w.md", source, nil)
	b := DeriveSpecBaseline(source, doc, planFor(t, doc, withdrawReg(t, true)))
	if got := DetectDrift(&b, doc, planFor(t, doc, withdrawReg(t, true))); len(got) != 0 {
		t.Errorf("expected no drift, got %v", got)
	}
}

func TestRenamedStepDriftsByName(t *testing.T) {
	source := "I withdraw 40."
	doc := Parse("w.md", source, nil)
	b := DeriveSpecBaseline(source, doc, planFor(t, doc, withdrawReg(t, true)))
	drift := DetectDrift(&b, doc, planFor(t, doc, withdrawReg(t, false)))
	if !reflect.DeepEqual(bare(drift), [][2]any{{"I withdraw 40", 1}}) {
		t.Errorf("got %v", bare(drift))
	}
}

func TestInPlaceTypoDriftsByLine(t *testing.T) {
	before := "I withdraw 40."
	beforeDoc := Parse("w.md", before, nil)
	b := DeriveSpecBaseline(before, beforeDoc, planFor(t, beforeDoc, withdrawReg(t, true)))
	afterDoc := Parse("w.md", "I withdrraw 40.", nil)
	drift := DetectDrift(&b, afterDoc, planFor(t, afterDoc, withdrawReg(t, true)))
	if !reflect.DeepEqual(bare(drift), [][2]any{{"I withdraw 40", 1}}) {
		t.Errorf("got %v", bare(drift))
	}
}

func TestDeletedParagraphNotDrift(t *testing.T) {
	before := "I withdraw 40."
	beforeDoc := Parse("w.md", before, nil)
	b := DeriveSpecBaseline(before, beforeDoc, planFor(t, beforeDoc, withdrawReg(t, true)))
	afterDoc := Parse("w.md", "", nil)
	if got := DetectDrift(&b, afterDoc, planFor(t, afterDoc, withdrawReg(t, true))); len(got) != 0 {
		t.Errorf("expected no drift, got %v", got)
	}
}

func TestMoveAndRewordStillMatchingNoDrift(t *testing.T) {
	before := "I withdraw 40.\n\nI withdraw 10."
	beforeDoc := Parse("w.md", before, nil)
	b := DeriveSpecBaseline(before, beforeDoc, planFor(t, beforeDoc, withdrawReg(t, true)))
	afterDoc := Parse("w.md", "I withdraw 11.\n\nI withdraw 40.", nil)
	if got := DetectDrift(&b, afterDoc, planFor(t, afterDoc, withdrawReg(t, true))); len(got) != 0 {
		t.Errorf("expected no drift, got %v", got)
	}
}

func TestRewrittenPastRecognitionNotDrift(t *testing.T) {
	before := "I withdraw 40."
	beforeDoc := Parse("w.md", before, nil)
	b := DeriveSpecBaseline(before, beforeDoc, planFor(t, beforeDoc, withdrawReg(t, true)))
	afterDoc := Parse("w.md", "The branch closed years ago.", nil)
	if got := DetectDrift(&b, afterDoc, planFor(t, afterDoc, withdrawReg(t, true))); len(got) != 0 {
		t.Errorf("expected no drift, got %v", got)
	}
}

func TestDriftDiagnosticsAreError(t *testing.T) {
	source := "I withdraw 40."
	doc := Parse("w.md", source, nil)
	b := DeriveSpecBaseline(source, doc, planFor(t, doc, withdrawReg(t, true)))
	diags := DriftDiagnostics(DetectDrift(&b, doc, planFor(t, doc, withdrawReg(t, false))))
	if len(diags) != 1 || diags[0].Severity != SeverityError || diags[0].Code != CodeDrift {
		t.Fatalf("diags = %v", diags)
	}
}

func TestReconcileRecordsThenReportsAndPreserves(t *testing.T) {
	source := "I withdraw 40."
	doc := Parse("w.md", source, nil)
	store := &memoryStore{}
	if got := ReconcileDrift(store, "w.md", source, doc, planFor(t, doc, withdrawReg(t, true)), false); len(got) != 0 {
		t.Fatalf("first run should have no drift, got %v", got)
	}
	before := store.contents
	drift := ReconcileDrift(store, "w.md", source, doc, planFor(t, doc, withdrawReg(t, false)), false)
	if !reflect.DeepEqual(bare(drift), [][2]any{{"I withdraw 40", 1}}) {
		t.Errorf("got %v", bare(drift))
	}
	if store.contents != before {
		t.Error("baseline should be untouched while drift unacknowledged")
	}
}

func TestReconcileUpdateModeAcceptsDrift(t *testing.T) {
	source := "I withdraw 40."
	doc := Parse("w.md", source, nil)
	store := &memoryStore{}
	ReconcileDrift(store, "w.md", source, doc, planFor(t, doc, withdrawReg(t, true)), false)
	drift := ReconcileDrift(store, "w.md", source, doc, planFor(t, doc, withdrawReg(t, false)), true)
	if len(drift) != 0 {
		t.Errorf("update mode should report no drift, got %v", drift)
	}
	lock := ParseVarLock(store.contents)
	if lock == nil || len(lock.Specs["w.md"].Examples) != 0 {
		t.Errorf("expected empty examples after update, got %v", lock)
	}
}

const expectedLock = `{
  "version": 1,
  "specs": {
    "library.md": {
      "sourceHash": "fnv1a:1a2b3c4d",
      "examples": [
        {
          "name": "I check out",
          "line": 7
        }
      ]
    }
  }
}
`

func TestStringifyMatchesSerializerByteForByte(t *testing.T) {
	lock := VarLock{Version: 1, Specs: map[string]SpecBaseline{
		"library.md": {SourceHash: "fnv1a:1a2b3c4d", Examples: []BaselineExample{{Name: "I check out", Line: 7}}},
	}}
	if got := StringifyVarLock(lock); got != expectedLock {
		t.Errorf("got:\n%s\nwant:\n%s", got, expectedLock)
	}
}

func TestParseRoundTrips(t *testing.T) {
	lock := VarLock{Version: 1, Specs: map[string]SpecBaseline{
		"library.md": {SourceHash: "fnv1a:1a2b3c4d", Examples: []BaselineExample{{Name: "I check out", Line: 7}}},
	}}
	got := ParseVarLock(StringifyVarLock(lock))
	if got == nil || !reflect.DeepEqual(*got, lock) {
		t.Errorf("round-trip = %v want %v", got, lock)
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	for _, bad := range []string{
		"not json",
		"{}",
		`{"version":2,"specs":{}}`,
		`{"version":1,"specs":{"a.md":{"examples":[]}}}`,
	} {
		if got := ParseVarLock(bad); got != nil {
			t.Errorf("ParseVarLock(%q) = %v, want nil", bad, got)
		}
	}
}
