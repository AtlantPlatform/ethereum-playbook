package model

type ReadCmds map[string]ReadCmdSpec

func (cmds *ReadCmds) Validate(ctx AppContext, spec *Spec) bool {
	return true
}

type ReadCmdSpec struct {
}
