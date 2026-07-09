package varrunner

import "github.com/oselvar/var/go/varcore"

// PlanSpec parses a spec and builds its ExecutionPlan against registry, also
// returning the VarDoc (needed for drift reconciliation). Port of plan_spec.
func PlanSpec(path, source string, registry varcore.Registry) (varcore.VarDoc, varcore.ExecutionPlan, error) {
	doc := varcore.Parse(path, source, nil)
	plan, err := varcore.BuildPlan(doc, registry)
	return doc, plan, err
}
