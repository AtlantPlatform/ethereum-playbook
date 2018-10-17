package model

import (
	"fmt"
	"math/big"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ethereum/go-ethereum/common"
)

type ParamSpec struct {
	Params []interface{} `yaml:"params"`

	paramValues []interface{} `yaml:"-"`
}

func (spec *ParamSpec) Validate(ctx AppContext, name string, root *Spec) bool {
	spec.paramValues = make([]interface{}, len(spec.Params))
	for paramID, param := range spec.Params {
		if !spec.validateParam(ctx, name, root, NewEvaler(), paramID, param) {
			return false
		}
	}
	// for k, v := range spec.paramValues {
	// 	log.Printf("%v: %#v %T", k, v, v)
	// }
	return true
}

func (spec *ParamSpec) ParamValues() []interface{} {
	return spec.paramValues
}

var PlaceholderAddr = common.BytesToAddress([]byte("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"))

func (spec *ParamSpec) validateParam(ctx AppContext,
	name string, root *Spec, evaler *Evaler, paramID int, param interface{}) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "ParamSpec",
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
		valueStr := nillableStr(p["value"])
		referenceStr := nillableStr(p["reference"])
		if len(valueStr) > 0 && len(referenceStr) > 0 {
			validateLog.Errorln("value and reference cannot co-exist in param spec", valueStr, referenceStr)
			return false
		}
		paramType := ParamType(typ.(string))

		if len(referenceStr) > 0 {
			refLog := validateLog.WithField("reference", referenceStr)
			if !isWalletRef(referenceStr) {
				refLog.Errorln("reference string is not a wallet field reference")
				return false
			}
			ref, err := newWalletFieldReference(ctx, root, referenceStr)
			if err != nil {
				refLog.WithError(err).Errorln("failed to resolve reference")
				return false
			}
			spec.paramValues[paramID] = ref
			return true
		}

		// allow cross-reference to the wallet section,
		// if param type is address, e.g. @bob
		if paramType == ParamTypeAddress {
			if isWalletRef(valueStr) {
				walletName := strings.TrimPrefix(valueStr, walletPrefix)
				if walletName == walletPrefix {
					spec.paramValues[paramID] = PlaceholderAddr // will be checked later
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

const (
	walletPrefix string = "@"
	refDelim     string = "."
)

func isWalletRef(str string) bool {
	return strings.HasPrefix(str, walletPrefix)
}

func isMathExp(str string) bool {
	for _, r := range str {
		if r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '+', '-', '*', '/', '.',
			'%', '&', '^', '(', ')', '<', '>', '=', ' ',
			'a', 'b', 'c', 'd', 'e', 'f',
			'A', 'B', 'C', 'D', 'E', 'F',
			'~', '|', 'x', 'X', 'p':
			continue
		}
		return false
	}
	return true
}

type ParamType string

const (
	ParamTypeString  ParamType = "string"
	ParamTypeInt     ParamType = "int"
	ParamTypeInt8    ParamType = "int8"
	ParamTypeInt16   ParamType = "int16"
	ParamTypeInt32   ParamType = "int32"
	ParamTypeInt64   ParamType = "int64"
	ParamTypeInt128  ParamType = "int128"
	ParamTypeInt256  ParamType = "int256"
	ParamTypeUInt    ParamType = "uint"
	ParamTypeUInt8   ParamType = "uint8"
	ParamTypeUInt16  ParamType = "uint16"
	ParamTypeUInt32  ParamType = "uint32"
	ParamTypeUInt64  ParamType = "uint64"
	ParamTypeUInt128 ParamType = "uint128"
	ParamTypeUInt256 ParamType = "uint256"
	ParamTypeBoolean ParamType = "bool"
	ParamTypeAddress ParamType = "address"
	ParamTypeByte    ParamType = "byte"
	ParamTypeBytes   ParamType = "bytes"
)

func parseParam(evaler *Evaler, typ ParamType, value string) (vv interface{}, ok bool) {
	parseIntBits := func(bits int) (interface{}, bool) {
		if result, err := evaler.Run(value, ExprTypeInterger); err == nil {
			tmp := result.(*big.Int)
			if bits == 0 || tmp.BitLen() <= bits {
				return tmp, true
			}
		} else {
			log.WithError(err).Warningln("param eval error")
			return nil, false
		}
		log.Warningln("param of incompatible type")
		return nil, false
	}
	parseUIntBits := func(bits int) (interface{}, bool) {
		if result, err := evaler.Run(value, ExprTypeInterger); err == nil {
			tmp := result.(*big.Int)
			if tmp.Sign() > 0 {
				if bits == 0 || tmp.BitLen() <= bits {
					return tmp, true
				}
			}
		} else {
			log.WithError(err).Warningln("param eval error")
			return nil, false
		}
		log.Warningln("param of incompatible type")
		return nil, false
	}
	switch typ {
	case ParamTypeString:
		vv = value
		ok = true
	case ParamTypeAddress:
		if ok = common.IsHexAddress(value); ok {
			vv = common.HexToAddress(value)
		}
	case ParamTypeByte:
		res, compatible := parseUIntBits(8)
		if compatible {
			vv = byte(res.(*big.Int).Uint64())
			ok = true
		}
	case ParamTypeBytes:
		vv = []byte(value)
		ok = true
	case ParamTypeBoolean:
		if result, err := evaler.Run(value, ExprTypeBool); err == nil {
			vv = result
			ok = true
		}
	case ParamTypeInt:
		vv, ok = parseIntBits(0)
	case ParamTypeUInt:
		vv, ok = parseUIntBits(0)
	case ParamTypeInt128:
		vv, ok = parseIntBits(128)
	case ParamTypeInt256:
		vv, ok = parseIntBits(256)
	case ParamTypeUInt128:
		vv, ok = parseUIntBits(128)
	case ParamTypeUInt256:
		vv, ok = parseUIntBits(256)
	case ParamTypeInt8:
		res, compatible := parseIntBits(8)
		if compatible {
			vv = int8(res.(*big.Int).Int64())
			ok = true
		}
	case ParamTypeInt16:
		res, compatible := parseIntBits(16)
		if compatible {
			vv = int16(res.(*big.Int).Int64())
			ok = true
		}
	case ParamTypeInt32:
		res, compatible := parseIntBits(32)
		if compatible {
			vv = int32(res.(*big.Int).Int64())
			ok = true
		}
	case ParamTypeInt64:
		res, compatible := parseIntBits(64)
		if compatible {
			vv = res.(*big.Int).Int64()
			ok = true
		}
	case ParamTypeUInt8:
		res, compatible := parseUIntBits(8)
		if compatible {
			vv = uint8(res.(*big.Int).Uint64())
			ok = true
		}
	case ParamTypeUInt16:
		res, compatible := parseUIntBits(16)
		if compatible {
			vv = uint16(res.(*big.Int).Uint64())
			ok = true
		}
	case ParamTypeUInt32:
		res, compatible := parseUIntBits(32)
		if compatible {
			vv = uint32(res.(*big.Int).Uint64())
			ok = true
		}
	case ParamTypeUInt64:
		res, compatible := parseUIntBits(64)
		if compatible {
			vv = uint64(res.(*big.Int).Uint64())
			ok = true
		}
	}
	return vv, ok
}

func nillableStr(str interface{}) string {
	if str == nil {
		return ""
	}
	return fmt.Sprintf("%v", str)
}
