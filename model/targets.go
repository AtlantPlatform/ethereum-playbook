package model

type Targets map[string]TargetSpec

func (targets *Targets) Validate(ctx AppContext, spec *Spec) bool {
	return true
}

type TargetSpec struct {
}
