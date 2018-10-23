package model

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
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
			typ = ParamTypeString
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
			if isWalletRef(referenceStr) {
				if referenceStr[1:] == walletPrefix {
					spec.paramValues[paramID] = PlaceholderAddr // will be resolved later
					return true
				}
				ref, err := newWalletFieldReference(root, referenceStr)
				if err != nil {
					refLog.WithError(err).Errorln("failed to resolve reference")
					return false
				}
				spec.paramValues[paramID] = ref // will be resolved later
				return true
			}
			referenceStrParts := strings.Split(referenceStr, " ")
			for i, part := range referenceStrParts {
				if isArgRef(part) {
					ref, err := newArgReference(ctx, part)
					if err != nil {
						refLog.WithError(err).Errorln("failed to resolve reference")
						return false
					}
					if ref.ArgID < 0 {
						// the whole param unresolved
						referenceStr = ""
						referenceStrParts = nil
						valueStr = ""
						break
					}
					referenceStrParts[i] = ctx.AppCommandArgs()[ref.ArgID]
				}
			}
			referenceStr = ""
			valueStr = strings.Join(referenceStrParts, " ")
			// continue with parsing
		} else if paramType == ParamTypeAddress {
			// allow cross-reference to the wallet section,
			// if param type is address, e.g. @bob
			if isWalletRef(valueStr) {
				walletName := valueStr[1:]
				if walletName == walletPrefix {
					spec.paramValues[paramID] = PlaceholderAddr // will be resolved later
					return true
				}
				if _, existing := root.Wallets.WalletSpec(walletName); !existing {
					validateLog.WithField("wallet", walletName).Errorln("unknown wallet reference")
					return false
				}
				ref := &WalletFieldReference{
					WalletName: walletName,
					FieldName:  WalletSpecAddressField,
				}
				spec.paramValues[paramID] = ref // will be resolved later
				return true
			}
		}
		if len(valueStr) > 0 {
			if v, ok := parseParam(evaler, paramType, valueStr); ok {
				spec.paramValues[paramID] = v
			} else {
				validateLog.WithFields(log.Fields{
					"offset": paramID,
					"type":   string(paramType),
					"value":  valueStr,
				}).Errorln("param parsing error, check type")
				return false
			}
		}
	case string:
		spec.paramValues[paramID] = param
	default:
		validateLog.Errorln("unsupported param type: expected string or object {type, value}")
		return false
	}
	return true
}

func (spec *ParamSpec) CountArgsUsing(set map[int]struct{}) {
	for _, param := range spec.Params {
		p, ok := param.(map[interface{}]interface{})
		if !ok {
			continue
		}
		referenceStr := nillableStr(p["reference"])
		if len(referenceStr) == 0 {
			continue
		}
		referenceStrParts := strings.Split(referenceStr, " ")
		for _, part := range referenceStrParts {
			if isArgRef(part) {
				if argID, err := argReferenceID(part); err == nil {
					set[argID] = struct{}{}
				}
			}
		}
	}
}

const (
	walletPrefix string = "@"
	argPrefix    string = "$"
	refDelim     string = "."
)

func isWalletRef(str string) bool {
	return strings.HasPrefix(str, walletPrefix)
}

func isArgRef(str string) bool {
	return strings.HasPrefix(str, argPrefix)
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
			if tmp.Sign() >= 0 {
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
		if strings.HasPrefix(value, "0x") {
			src := []byte(value[2:])
			dst := make([]byte, len(src)/2)
			if _, err := hex.Decode(dst, src); err != nil {
				ok = false
				return
			}
			vv = dst
			ok = true
		} else {
			vv = []byte(value)
			ok = true
		}
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
			vv = res.(*big.Int).Uint64()
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

func newArgReference(ctx AppContext, value string) (*ArgReference, error) {
	argID, err := strconv.Atoi(value[1:])
	if err != nil {
		err := errors.New("reference must be of the form $0, $1, etc")
		return nil, err
	}
	args := ctx.AppCommandArgs()
	if argID > len(args)-1 {
		noRef := &ArgReference{
			ArgID: -1,
		}
		return noRef, nil
	}
	ref := &ArgReference{
		ArgID: argID,
	}
	return ref, nil
}

func argReferenceID(value string) (int, error) {
	argID, err := strconv.Atoi(value[1:])
	if err != nil {
		err := errors.New("reference must be of the form $0, $1, etc")
		return -1, err
	}
	return argID, nil
}

type ArgReference struct {
	ArgID int
}
