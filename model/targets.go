package model

import log "github.com/Sirupsen/logrus"

type Targets map[string]TargetSpec

func (targets Targets) Validate(ctx AppContext, spec *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"Section": "Targets",
		"func":    "Validate",
	})
	for name, target := range targets {
		if _, ok := spec.uniqueNames[name]; ok {
			validateLog.WithField("name", name).Errorln("target name is not unique")
			return false
		} else {
			spec.uniqueNames[name] = struct{}{}
		}
		if !target.Validate(ctx, name, spec) {
			return false
		}
	}
	return true
}

type TargetSpec struct {
}

func (spec *TargetSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	return true
}
