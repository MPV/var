package varcore

// step_role.go — the role a step definition plays. Port of step_role.py /
// step-role.ts.
//
//   - Stimulus drives the software: arranges the quiescent state AND acts on it.
//   - Sensor is the read-only assertion (the only role that returns for
//     comparison).
//
// The arrange/act (given/when) concepts remain useful narration in a document,
// but they share one mechanism: a stimulus evolves state, a sensor observes it.
type StepKind string

const (
	Stimulus StepKind = "stimulus"
	Sensor   StepKind = "sensor"
)

// InferStepRole guesses a step's role from its document-order neighbours.
// Purely structural — never inspects sentence words (no Given/When/Then
// heuristics). A step with nothing after it is most likely the observation;
// anything followed by other steps is most likely driving the software. Port of
// infer_step_role.
func InferStepRole(after int) StepKind {
	if after == 0 {
		return Sensor
	}
	return Stimulus
}
