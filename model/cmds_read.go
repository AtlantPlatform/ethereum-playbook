package model

import (
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type ReadCmds map[string]*ReadCmdSpec

func (cmds ReadCmds) Validate(ctx AppContext, spec *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "ReadCmds",
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

func (cmds ReadCmds) ReadCmdSpec(name string) (*ReadCmdSpec, bool) {
	spec, ok := cmds[name]
	return spec, ok
}

type ReadCmdSpec struct {
	ParamSpec   `yaml:",inline"`
	Description string `yaml:"desc"`

	Wallet string `yaml:"wallet"`
	Method string `yaml:"method"`

	Instance *ContractInstanceSpec `yaml:"instance"`

	walletRx *regexp.Regexp `yaml:"-"`
	matching []*WalletSpec  `yaml:"-"`
}

func (spec *ReadCmdSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "ReadCommands",
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
	if spec.Instance == nil {
		validateLog.Errorln("no target contract instance specified")
		return false
	} else if len(spec.Instance.Name) == 0 {
		validateLog.Errorln("the target contract spec name is not specified")
		return false
	}
	contract, ok := root.Contracts.ContractSpec(spec.Instance.Name)
	if !ok || contract == nil {
		validateLog.Errorln("the target contract spec not found (name mismatch)")
		return false
	} else if len(contract.Instances) == 0 {
		validateLog.Errorln("the target contract spec has no instances")
		return false
	}
	address := strings.ToLower(spec.Instance.Address)
	if len(address) == 0 {
		spec.Instance = contract.Instances[0]
	} else {
		var found bool
		for _, instance := range contract.Instances {
			if instance.Address == address {
				found = true
				spec.Instance = instance
				break
			}
		}
		if !found {
			validateLog.Errorln("referenced contract instance is not found (address mismatch)")
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

func (spec *ReadCmdSpec) MatchingWallets() []*WalletSpec {
	return spec.matching
}

func (spec *ReadCmdSpec) CountArgsUsing(set map[int]struct{}) {
	spec.ParamSpec.CountArgsUsing(set)
}

func (spec *ReadCmdSpec) ArgCount() int {
	set := make(map[int]struct{})
	spec.CountArgsUsing(set)
	return len(set)
}
