package model

import (
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type WriteCmds map[string]*WriteCmdSpec

func (cmds WriteCmds) Validate(ctx AppContext, spec *Spec) bool {
	for name, cmd := range cmds {
		if !cmd.Validate(ctx, name, spec) {
			return false
		}
	}
	return true
}

func (cmds WriteCmds) WriteCmdSpec(name string) (*WriteCmdSpec, bool) {
	spec, ok := cmds[name]
	return spec, ok
}

type WriteCmdSpec struct {
	ParamSpec `yaml:",inline"`

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
	if rx, err := regexp.Compile(spec.Wallet); err != nil {
		validateLog.WithError(err).Errorln("failed to compile wallet regexp")
		return false
	} else {
		spec.walletRx = rx
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
		if spec.Instance.IsDeployed() && len(spec.Method) == 0 {
			validateLog.Errorln("the contract is deployed, but no recipient method specified")
			return false
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
		if spec.To != "0x0" {
			if wallet, ok := root.Wallets.WalletSpec(spec.To); !ok {
				validateLog.Errorln("recipient 'to' wallet name is not found")
				return false
			} else if wallet.Address == "" || wallet.Address == "0x0" {
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

// 	validDenominations := append([]string{}, commonDenominations...)
// 	for _, contract := range root.Contracts {
// 		for _, instance := range contract.Instances {
// 			if len(instance.tokenSymbol) > 0 {
// 				validDenominations = append(validDenominations, strings.ToLower(instance.tokenSymbol))
// 			}
// 		}
// 	}
