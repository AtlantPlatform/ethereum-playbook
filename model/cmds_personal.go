package model

type PersonalCmds map[string]PersonalCmdSpec

func (cmds *PersonalCmds) Validate(ctx AppContext, spec *Spec) bool {
	return true
}

type PersonalCmdSpec struct {
}
