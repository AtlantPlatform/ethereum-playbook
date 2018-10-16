package model

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ethereum/go-ethereum/common"
)

type ReadCmds map[string]*ReadCmdSpec

func (cmds ReadCmds) Validate(ctx AppContext, spec *Spec) bool {
	for name, cmd := range cmds {
		if !cmd.Validate(ctx, name, spec) {
			return false
		}
	}
	return true
}

type ReadCmdSpec struct {
	Wallet string `yaml:"wallet"`
	Method string `yaml:"method"`

	Params   []interface{}         `yaml:"params"`
	Instance *ContractInstanceSpec `yaml:"instance"`

	walletRx    *regexp.Regexp `yaml:"-"`
	matching    []*WalletSpec  `yaml:"-"`
	paramValues []interface{}  `yaml:"-"`
}

func (spec *ReadCmdSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "ReadCommands",
		"command": name,
	})
	var hasWalletName bool
	if len(spec.Wallet) > 0 {
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
	spec.matching = root.Wallets.GetAll(spec.walletRx)
	if hasWalletName {
		if len(spec.matching) == 0 {
			validateLog.Errorln("no wallets are matching the specified regexp")
			return false
		}
	}
	if len(spec.Method) == 0 {
		validateLog.Errorln("no method name is specified")
		return false
	}
	spec.paramValues = make([]interface{}, len(spec.Params))
	for paramID, param := range spec.Params {
		if !spec.validateParam(ctx, name, root, NewEvaler(), paramID, param) {
			return false
		}
	}
	for k, v := range spec.paramValues {
		log.Printf("%v: %#v %T", k, v, v)
	}
	return true
}

var placeholderAddr = common.BytesToAddress([]byte("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"))

func (spec *ReadCmdSpec) validateParam(ctx AppContext,
	name string, root *Spec, evaler *Evaler, paramID int, param interface{}) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "ReadCommands",
		"command": name,
	})
	switch p := param.(type) {
	case map[interface{}]interface{}:
		typ, ok := p["type"]
		if !ok {
			// default to string
			spec.paramValues[paramID] = param
			return true
		}
		valueStr := fmt.Sprintf("%v", p["value"])
		paramType := ParamType(typ.(string))

		// allow cross-reference to the wallet section,
		// if param type is address, e.g. @bob
		if paramType == ParamTypeAddress {
			if isWalletRef(valueStr) {
				walletName := strings.TrimPrefix(valueStr, walletPrefix)
				if walletName == walletPrefix {
					spec.paramValues[paramID] = placeholderAddr // will be checked later
					return true
				}
				if wallet, existing := root.Wallets.WalletSpec(walletName); !existing {
					validateLog.WithField("wallet", walletName).Errorln("unknown wallet reference")
					return false
				} else {
					spec.paramValues[paramID] = common.HexToAddress(wallet.Address)
					return true
				}
			}
		}

		v, ok := parseParam(evaler, paramType, valueStr)
		if !ok {
			validateLog.WithFields(log.Fields{
				"offset": paramID,
				"type":   string(paramType),
				"value":  valueStr,
			}).Errorln("param parsing error, check type")
			return false
		}
		spec.paramValues[paramID] = v
	case string:
		spec.paramValues[paramID] = param
	default:
		validateLog.Println("unsupported param type: expected string or object {type, value}")
		return false
	}
	return true
}
