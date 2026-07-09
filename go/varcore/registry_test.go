package varcore

import (
	"reflect"
	"testing"
)

func TestParameterTypeNames(t *testing.T) {
	cases := map[string][]string{
		"I have {int} cukes":            {"int"},
		"The square of {int} is {int}.": {"int", "int"},
		"I always boom":                 {},
		`escaped \{int\} literal`:       {},
		"The late fee is {money}":       {"money"},
	}
	for expr, want := range cases {
		got := parameterTypeNames(expr)
		if len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("parameterTypeNames(%q) = %v, want %v", expr, got, want)
		}
	}
}

func TestToRegistryArtifactListsExpressionsAndNames(t *testing.T) {
	r := CreateRegistry()
	r, err := AddStep(r, "I have {int} cukes", "s.go", 1, nil, Stimulus)
	if err != nil {
		t.Fatal(err)
	}
	art := ToRegistryArtifact(r)
	if got := art["parameterTypes"].([]any); len(got) != 0 {
		t.Errorf("parameterTypes = %v, want empty", got)
	}
	steps := art["steps"].([]any)
	if len(steps) != 1 {
		t.Fatalf("steps len = %d, want 1", len(steps))
	}
	step := steps[0].(map[string]any)
	if step["expression"] != "I have {int} cukes" {
		t.Errorf("expression = %v", step["expression"])
	}
	if names := step["parameterTypeNames"].([]any); len(names) != 1 || names[0] != "int" {
		t.Errorf("parameterTypeNames = %v", names)
	}
}

func TestAddStepRejectsDuplicate(t *testing.T) {
	r := CreateRegistry()
	r, _ = AddStep(r, "I increment", "a.go", 1, nil, Stimulus)
	if _, err := AddStep(r, "I increment", "b.go", 2, nil, Stimulus); err == nil {
		t.Error("expected duplicate error")
	}
}

func TestDefineParameterTypeArtifact(t *testing.T) {
	r := CreateRegistry()
	r = DefineParameterType(r, CustomParamType{Name: "airport", Regexp: "[A-Z]{3}"}, nil)
	r, _ = AddStep(r, "I fly to {airport}", "s.go", 1, nil, Stimulus)
	art := ToRegistryArtifact(r)
	pts := art["parameterTypes"].([]any)
	if len(pts) != 1 {
		t.Fatalf("parameterTypes len = %d", len(pts))
	}
	pt := pts[0].(map[string]any)
	if pt["name"] != "airport" || pt["regexp"] != "[A-Z]{3}" {
		t.Errorf("custom param type = %v", pt)
	}
}
