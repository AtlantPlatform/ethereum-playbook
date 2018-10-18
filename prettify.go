package main

import (
	"fmt"
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
	case *hexutil.Big:
		return vv.ToInt().String()
	case common.Address:
		return strings.ToLower(vv.Hex())
	case bool:
		return vv
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v (%T)", vv, vv)
	}
	return ""
}

func prettify(v interface{}) interface{} {
	switch vv := v.(type) {
	case []interface{}:
		for i, vvv := range vv {
			vv[i] = prettifyValue(vvv)
		}
	case map[string]interface{}:
		container := make(map[string]interface{})
		for key, vvv := range vv {
			container[key] = prettifyValue(vvv)
		}
		return container
	default:
		return prettifyValue(v)
	}
	return ""
}
