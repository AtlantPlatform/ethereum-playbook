package model

import (
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Targets map[string]TargetSpec

func (targets Targets) Validate(ctx AppContext, spec *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "Targets",
		"func":    "Validate",
	})
	for name, target := range targets {
		if _, ok := spec.uniqueNames[name]; ok {
			validateLog.WithField("name", name).Errorln("target name is not unique")
			return false
		}
		spec.uniqueNames[name] = struct{}{}

		if !target.Validate(ctx, name, spec) {
			return false
		}
	}
	return true
}

func (targets Targets) TargetSpec(name string) (TargetSpec, bool) {
	spec, ok := targets[name]
	return spec, ok
}

type TargetSpec []TargetCommandSpec

func (spec TargetSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "Targets",
		"target":  "Validate",
	})
	for _, cmdSpec := range spec {
		cmdName := cmdSpec.Name()
		var found bool
		if _, found = root.CallCmds[cmdName]; found {
			if cmdSpec.IsDeferred() {
				validateLog.WithField("command", cmdName).Errorln("call commands are deferred by default")
				return false
			}
			continue
		}
		if _, found = root.ReadCmds[cmdName]; found {
			if cmdSpec.IsDeferred() {
				validateLog.WithField("command", cmdName).Errorln("read commands are deferred by default")
				return false
			}
			continue
		}
		if _, found = root.WriteCmds[cmdName]; found {
			continue
		}
		if !found {
			validateLog.WithField("command", cmdName).Errorln("command from target not found")
			return false
		}
	}
	return true
}

func (spec TargetSpec) CmdNames() []string {
	names := make([]string, 0, len(spec))
	for _, cmd := range spec {
		names = append(names, cmd.Name())
	}
	return names
}

func (spec TargetSpec) ArgCount(root *Spec) int {
	set := make(map[int]struct{})
	for _, cmd := range spec {
		root.CountArgsUsing(set, cmd.Name())
	}
	return len(set)
}

type TargetCommandSpec string

const targetCommandDefer = "&"

func (spec TargetCommandSpec) Name() string {
	return strings.TrimSpace(strings.TrimSuffix(string(spec), targetCommandDefer))
}

func (spec TargetCommandSpec) IsDeferred() bool {
	return strings.HasSuffix(string(spec), targetCommandDefer)
}
