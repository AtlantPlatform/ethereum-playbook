package model

import (
	"math/big"

	"github.com/AtlantPlatform/ethfw"
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

type WriteCmdSpec struct {
}

func (spec *WriteCmdSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	// validateLog := log.WithFields(log.Fields{
	// 	"section": "WriteCommands",
	// 	"command": name,
	// })
	// 	var hasWalletName bool
	// 	if len(spec.Wallet) > 0 {
	// 		hasWalletName = true
	// 	} else {
	// 		// match all by default
	// 		spec.Wallet = "."
	// 	}
	// 	if rx, err := regexp.Compile(spec.Wallet); err != nil {
	// 		validateLog.WithError(err).Errorln("failed to compile wallet regexp")
	// 		return false
	// 	} else {
	// 		spec.walletRx = rx
	// 	}
	// 	spec.matching = root.Wallets.GetAll(spec.walletRx)
	// 	if hasWalletName {
	// 		if len(spec.matching) == 0 {
	// 			validateLog.Errorln("no wallets are matching the specified regexp")
	// 			return false
	// 		}
	// 	}
	// 	if len(spec.Method) == 0 {
	// 		validateLog.Errorln("no method name is specified")
	// 		return false
	// 	}
	// 	mathEval := NewEvaler()
	// 	validDenominations := append([]string{}, commonDenominations...)
	// 	for _, contract := range root.Contracts {
	// 		for _, instance := range contract.Instances {
	// 			if len(instance.tokenSymbol) > 0 {
	// 				validDenominations = append(validDenominations, strings.ToLower(instance.tokenSymbol))
	// 			}
	// 		}
	// 	}
	// 	spec.paramValues = make([]interface{}, 0, len(spec.Params))
	// 	for paramID, param := range spec.Params {
	// 		if isWalletRef(param) {
	// 			walletName := strings.TrimPrefix(param, walletPrefix)
	// 			if walletName == walletPrefix {
	// 				spec.paramValues[paramID] = walletPrefix
	// 				continue
	// 			}
	// 			if wallet, existing := root.Wallets.WalletSpec(walletName); !existing {
	// 				validateLog.WithField("wallet", walletName).Errorln("unknown wallet reference")
	// 				return false
	// 			} else {
	// 				spec.paramValues[paramID] = wallet.Address
	// 				continue
	// 			}
	// 		}
	// 		if param == "true" {
	// 			spec.paramValues[paramID] = true
	// 			continue
	// 		} else if param == "false" {
	// 			spec.paramValues[paramID] = false
	// 			continue
	// 		}
	// 		param = strings.ToLower(param)

	// 		var isValueSpec bool
	// 		var valueDenomintator string
	// 		for _, den := range validDenominations {
	// 			if strings.HasSuffix(param, den) {
	// 				expr := strings.TrimSuffix(param, den)
	// 				if isMathExp(expr) {
	// 					isValueSpec = true
	// 					param = expr
	// 					valueDenomintator = den
	// 					break
	// 				}
	// 			}
	// 		}
	// 		if isValueSpec || isMathExp(param) {
	// 			evalLog := validateLog.WithField("expr", param)
	// 			value, err := mathEval.Run(param, ExprTypeInterger)
	// 			if err != nil {
	// 				evalLog.WithError(err).Errorln("failed to evaluate math expression")
	// 				return false
	// 			}
	// 			if isValueSpec {
	// 				value = convertValue(value, valueDenomintator)
	// 			}
	// 			spec.paramValues[paramID] = value
	// 			continue
	// 		}
	// 		// at this point, it is considered as a simple string
	// 		spec.paramValues[paramID] = param
	// 	}
	return true
}

func convertValue(val *big.Int, denominator string) *big.Int {
	switch denominator {
	case "wei", "ether":
		return val
	case "gwei":
		return ethfw.BigWei(val).Mul(1e9).ToInt()
	case "eth":
		return ethfw.ToWei(float64(val.Int64())).ToInt()
	default:
		return val
	}
}

var commonDenominations = []string{
	"wei", "gwei", "ether",
	"eth",
}
