package main

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func prettifyValue(v interface{}) interface{} {
	switch vv := v.(type) {
	case string:
		if strings.HasPrefix(vv, "tx:") {
			return vv[3:]
		}
		if strings.HasPrefix(vv, "0x") {
			if common.IsHexAddress(vv) {
				return strings.ToLower(vv)
			}
			vvv, err := hexutil.DecodeBig(vv)
			if err != nil {
				return vv
			} else if vvv.BitLen() > 256 {
				return vv
			}
			return vvv.String()
		}
		return vv
	case *big.Int:
		return vv.String()
	case *hexutil.Big:
		return vv.ToInt().String()
	case common.Address:
		return strings.ToLower(vv.Hex())
	case bool:
		return vv
	case int:
		return vv
	case int8:
		return vv
	case int16:
		return vv
	case int32:
		return vv
	case int64:
		return vv
	case uint:
		return vv
	case uint8:
		return vv
	case uint16:
		return vv
	case uint32:
		return vv
	case uint64:
		return vv
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v (%T)", vv, vv)
	}
}

func prettify(v interface{}) interface{} {
	switch vv := v.(type) {
	case []interface{}:
		formatted := make([]interface{}, len(vv))
		for i, vvv := range vv {
			formatted[i] = prettifyValue(vvv)
		}
		return formatted
	case map[string]interface{}:
		container := make(map[string]interface{})
		for key, vvv := range vv {
			container[key] = prettifyValue(vvv)
		}
		return container
	default:
		return prettifyValue(v)
	}
}
