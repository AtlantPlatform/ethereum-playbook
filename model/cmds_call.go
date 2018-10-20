package model

import (
	"regexp"

	log "github.com/Sirupsen/logrus"
)

type CallCmds map[string]*CallCmdSpec

func (cmds CallCmds) Validate(ctx AppContext, spec *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "CallCmds",
		"func":    "Validate",
	})
	for name, cmd := range cmds {
		if _, ok := spec.uniqueNames[name]; ok {
			validateLog.WithField("name", name).Errorln("cmd name is not unique")
			return false
		} else {
			spec.uniqueNames[name] = struct{}{}
		}
		if !cmd.Validate(ctx, name, spec) {
			return false
		}
	}
	return true
}

func (cmds CallCmds) CallCmdSpec(name string) (*CallCmdSpec, bool) {
	spec, ok := cmds[name]
	return spec, ok
}

type CallCmdSpec struct {
	ParamSpec   `yaml:",inline"`
	Description string `yaml:"desc"`

	Wallet string `yaml:"wallet"`
	Method string `yaml:"method"`

	walletRx *regexp.Regexp `yaml:"-"`
	matching []*WalletSpec  `yaml:"-"`
}

func (spec *CallCmdSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "CallCommands",
		"command": name,
	})
	var hasWalletName bool
	if len(spec.Wallet) > 0 {
		if isWalletRef(spec.Wallet) {
			validateLog.Errorln("wallet reference is not allowed in 'wallet' field, must be name")
			return false
		}
		hasWalletName = true
	}
	if rx, err := regexp.Compile(spec.Wallet); err != nil {
		validateLog.WithError(err).Errorln("failed to compile wallet regexp")
		return false
	} else {
		spec.walletRx = rx
	}
	if hasWalletName {
		spec.matching = root.Wallets.GetAll(spec.walletRx)
		if len(spec.matching) == 0 {
			validateLog.Errorln("no wallets are matching the specified regexp")
			return false
		}
	}
	if len(spec.Method) == 0 {
		validateLog.Errorln("no method name is specified")
		return false
	}
	if !spec.ParamSpec.Validate(ctx, name, root) {
		return false
	}
	return true
}

func (spec *CallCmdSpec) MatchingWallets() []*WalletSpec {
	return spec.matching
}

func (spec *CallCmdSpec) CountArgsUsing(set map[int]struct{}) {
	spec.ParamSpec.CountArgsUsing(set)
}

func (spec *CallCmdSpec) ArgCount() int {
	set := make(map[int]struct{})
	spec.CountArgsUsing(set)
	return len(set)
}
