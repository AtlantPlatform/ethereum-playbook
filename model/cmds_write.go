package model

type WriteCmds map[string]WriteCmdSpec

func (cmds *WriteCmds) Validate(ctx AppContext, spec *Spec) bool {
	return true
}

type WriteCmdSpec struct {
}
