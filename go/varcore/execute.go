package varcore

import (
	"reflect"
	"strings"
)

// execute.go — execute an ExecutionPlan: invoke handlers (via reflection),
// evolve state by full replacement (ADR 0006 — the Go fork from TS/Python's
// partial-merge), compare sensor returns via the diff helpers, and invert
// expected-failure outcomes. Port of execute.py / execute.ts adapted to Go's
// value semantics (no deep-freeze needed) and error-return control flow (no
// exceptions).

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// UnexpectedPassError is returned when an expected-to-fail example passes.
type UnexpectedPassError struct{}

func (e *UnexpectedPassError) Error() string {
	return "expected the example to fail, but it passed"
}

func isUnexpectedPassError(e error) bool {
	_, ok := e.(*UnexpectedPassError)
	return ok
}

// StepObservation is a per-step outcome for the trace.
type StepObservation struct {
	Ordinal int
	Outcome string // "pass" | "fail"
	Err     error
}

// callHandler invokes a step handler via reflection. The first parameter is the
// state; the remaining parameters receive the leading captures (then any
// trailing data table / doc string), converted to the handler's parameter
// types. The handler's error return (if any) becomes retErr; any other return is
// retVal.
func callHandler(handler any, state any, args, extra []any) (retVal any, retErr error) {
	hv := reflect.ValueOf(handler)
	ht := hv.Type()
	numIn := ht.NumIn()
	inputs := append(append([]any{}, args...), extra...)
	want := numIn - 1

	in := make([]reflect.Value, numIn)
	in[0] = convertArg(state, ht.In(0))
	for k := 0; k < want; k++ {
		var v any
		if k < len(inputs) {
			v = inputs[k]
		}
		in[k+1] = convertArg(v, ht.In(k+1))
	}

	for _, rv := range hv.Call(in) {
		if rv.Type().Implements(errorType) {
			if !rv.IsNil() {
				retErr, _ = rv.Interface().(error)
			}
		} else {
			retVal = rv.Interface()
		}
	}
	return
}

func convertArg(v any, t reflect.Type) reflect.Value {
	if v == nil {
		return reflect.Zero(t)
	}
	rv := reflect.ValueOf(v)
	if rv.Type().AssignableTo(t) {
		return rv
	}
	if rv.Type().ConvertibleTo(t) {
		return rv.Convert(t)
	}
	return rv
}

// executeExample runs one planned example, emitting a StepObservation per step,
// and returns the example's final error (nil = pass), with expected-failure
// inversion applied. Port of the run() closure in execute_plan.
func executeExample(plan ExecutionPlan, ex PlannedExample, createContext func(string) any, observe func(StepObservation)) error {
	stateByFile := map[string]any{}
	var lastReturn any
	var thrown error
	source := plan.VarDoc.Source

	for i, step := range ex.Steps {
		file := step.StepDef.ExpressionSourceFile
		if _, ok := stateByFile[file]; !ok {
			stateByFile[file] = createContext(file)
		}
		state := stateByFile[file]

		var extra []any
		if step.DataTable != nil {
			tbl := [][]string{append([]string{}, step.DataTable.Header.Cells...)}
			for _, row := range step.DataTable.Rows {
				tbl = append(tbl, append([]string{}, row.Cells...))
			}
			extra = append(extra, tbl)
		} else if step.DocString != nil {
			extra = append(extra, step.DocString.Content)
		}

		retVal, retErr := callHandler(step.StepDef.Handler, state, step.Args, extra)
		if retErr != nil {
			observe(StepObservation{Ordinal: i + 1, Outcome: "fail", Err: retErr})
			thrown = retErr
			break
		}
		lastReturn = retVal

		var err error
		switch step.StepDef.Kind {
		case Stimulus:
			// Full replacement: a stimulus returns the whole next state.
			if retVal != nil {
				state = retVal
				stateByFile[file] = state
			}
		case Sensor:
			if ex.RowChecks == nil && retVal != nil {
				err = applySlotContract(source, step, retVal, len(extra))
			}
		default:
			err = &ReturnShapeError{Message: "unknown step kind"}
		}

		if err != nil {
			observe(StepObservation{Ordinal: i + 1, Outcome: "fail", Err: err})
			thrown = err
			break
		}
		observe(StepObservation{Ordinal: i + 1, Outcome: "pass"})
	}

	// Header-bound row checks run after all steps complete.
	if thrown == nil && ex.RowChecks != nil {
		bad := filterNotOK(compareRow(lastReturn, ex.RowChecks))
		if len(bad) > 0 {
			cmErr := &CellMismatchError{Cells: bad}
			observe(StepObservation{Ordinal: len(ex.Steps), Outcome: "fail", Err: cmErr})
			thrown = cmErr
		}
	}

	// Expected-failure inversion.
	if ex.ExpectedOutcome == "fail" {
		if thrown == nil {
			return &UnexpectedPassError{}
		}
		if ex.ExpectedErrorMessage != nil && !strings.Contains(thrown.Error(), *ex.ExpectedErrorMessage) {
			return thrown
		}
		return nil
	}
	return thrown
}

// applySlotContract implements the sensor slot contract (zero/one/two+ slots)
// and runs the appropriate comparison. extraCount is 1 if a data table or doc
// string is attached, else 0.
func applySlotContract(source string, step PlannedStep, retVal any, extraCount int) error {
	slotCount := len(step.Args) + extraCount
	if slotCount == 0 {
		return &ReturnShapeError{Message: "this sensor has no parameters, data table or doc string — nothing to compare a return value against (raise to fail, return nothing to pass)"}
	}

	var slots []any
	if slotCount == 1 {
		slots = []any{retVal}
	} else {
		rv := reflect.ValueOf(retVal)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return &ReturnShapeError{Message: "a sensor with multiple parameters must return a list"}
		}
		if rv.Len() != slotCount {
			return &ReturnShapeError{Message: "sensor return has the wrong number of elements"}
		}
		slots = make([]any, rv.Len())
		for i := range slots {
			slots[i] = rv.Index(i).Interface()
		}
	}

	inlineReturned := slots[:len(step.Args)]
	sourceTexts := make([]string, len(step.ParamSpans))
	for i, sp := range step.ParamSpans {
		sourceTexts[i] = utf16Slice(source, sp.StartOffset, sp.EndOffset)
	}
	paramDiffs := filterNotOK(compareParams(inlineReturned, step.Args, step.ParamSpans, sourceTexts, step.Formats))
	if len(paramDiffs) > 0 {
		return &CellMismatchError{Cells: paramDiffs}
	}

	if step.DataTable != nil {
		bad, err := compareTable(slots[len(step.Args)], *step.DataTable)
		if err != nil {
			return err
		}
		if b := filterNotOK(bad); len(b) > 0 {
			return &CellMismatchError{Cells: b}
		}
	} else if step.DocString != nil {
		diff, err := compareDocString(slots[len(step.Args)], step.DocString.Content, step.DocString.Span)
		if err != nil {
			return err
		}
		if diff != nil {
			return &DocStringMismatchError{Diff: *diff}
		}
	}
	return nil
}

func filterNotOK(diffs []CellDiff) []CellDiff {
	out := make([]CellDiff, 0, len(diffs))
	for _, d := range diffs {
		if !d.OK {
			out = append(out, d)
		}
	}
	return out
}
