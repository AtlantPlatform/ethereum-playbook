package model

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type WriteCmds map[string]*WriteCmdSpec

func (cmds WriteCmds) Validate(ctx AppContext, spec *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "WriteCmds",
		"func":    "Validate",
	})
	for name, cmd := range cmds {
		if _, ok := spec.uniqueNames[name]; ok {
			validateLog.WithField("name", name).Errorln("cmd name is not unique")
			return false
		}
		spec.uniqueNames[name] = struct{}{}

		if ctx.AppCommand() == name {
			if !cmd.Validate(ctx, name, spec) {
				return false
			}
		}
	}
	return true
}

func (cmds WriteCmds) WriteCmdSpec(name string) (*WriteCmdSpec, bool) {
	spec, ok := cmds[name]
	return spec, ok
}

type WriteCmdSpec struct {
	ParamSpec   `yaml:",inline"`
	Description string `yaml:"desc"`

	Wallet string `yaml:"wallet"`
	Sticky string `yaml:"sticky"`
	To     string `yaml:"to"`
	Value  Valuer `yaml:"value"`
	Method string `yaml:"method"`

	Instance *ContractInstanceSpec `yaml:"instance"`

	walletRx *regexp.Regexp `yaml:"-"`
	matching *WalletSpec    `yaml:"-"`
}

func (spec *WriteCmdSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "WriteCommands",
		"command": name,
	})
	var hasWalletName bool
	if len(spec.Wallet) > 0 {
		if isWalletRef(spec.Wallet) {
			validateLog.Errorln("wallet reference is not allowed in 'wallet' field, must be name")
			return false
		}
		hasWalletName = true
	} else {
		// match all by default
		spec.Wallet = "."
	}
	rx, err := regexp.Compile(spec.Wallet)
	if err != nil {
		validateLog.WithError(err).Errorln("failed to compile wallet regexp")
		return false
	}
	spec.walletRx = rx

	if len(spec.Sticky) == 0 {
		spec.Sticky = name
	}
	spec.matching = root.Wallets.GetOne(spec.walletRx, spec.Sticky)
	if hasWalletName {
		if spec.matching == nil {
			validateLog.Errorln("no wallets are matching the specified regexp")
			return false
		}
	} else {
		validateLog.Errorln("no wallets specified to send from")
		return false
	}
	if len(spec.To) == 0 {
		if spec.Instance == nil {
			validateLog.Errorln("no recipient contract instance specified")
			return false
		} else if len(spec.Instance.Name) == 0 {
			validateLog.Errorln("the recipient contract spec name is not specified")
			return false
		}
		contract, ok := root.Contracts.ContractSpec(spec.Instance.Name)
		if !ok || contract == nil {
			validateLog.Errorln("the recipient contract spec not found (name mismatch)")
			return false
		} else if len(contract.Instances) == 0 {
			validateLog.Errorln("the recipient contract spec has no instances")
			return false
		}
		address := strings.ToLower(spec.Instance.Address)
		if len(address) == 0 {
			spec.Instance = contract.Instances[0]
		} else {
			var found bool
			for _, instance := range contract.Instances {
				if strings.ToLower(instance.Address) == address {
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
		// TODO: check ABI
	} else if spec.Instance != nil {
		validateLog.Errorln("contract instance must not be specified while using recipient 'to' address")
		return false
	} else {
		if isWalletRef(spec.To) {
			validateLog.Errorln("wallet reference is not allowed in 'to' field, must be name")
			return false
		}
		if spec.To != ZeroAddress {
			if wallet, ok := root.Wallets.WalletSpec(spec.To); !ok {
				validateLog.Errorln("recipient 'to' wallet name is not found")
				return false
			} else if wallet.Address == "" || wallet.Address == ZeroAddress {
				validateLog.Errorln("recipient 'to' wallet has no address. For 0x0, use '0x0' instead of name")
				return false
			} else {
				spec.To = wallet.Address
			}
		}
	}
	if !spec.ParamSpec.Validate(ctx, name, root) {
		return false
	}
	return true
}

func (spec *WriteCmdSpec) MatchingWallet() *WalletSpec {
	return spec.matching
}

func (spec *WriteCmdSpec) CountArgsUsing(set map[int]struct{}) {
	spec.ParamSpec.CountArgsUsing(set)
	spec.Value.CountArgsUsing(set)
}

func (spec *WriteCmdSpec) ArgCount() int {
	set := make(map[int]struct{})
	spec.CountArgsUsing(set)
	return len(set)
}
