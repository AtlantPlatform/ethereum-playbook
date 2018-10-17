package model

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/AtlantPlatform/ethfw"
)

type Valuer string

func (v Valuer) Parse(ctx AppContext, root *Spec, additionalDenominators []string) (*ExtendedValue, error) {
	valueStr := string(v)
	valueStrParts := strings.Split(valueStr, " ")
	for i, part := range valueStrParts {
		if isWalletRef(part) {
			ref, err := newWalletFieldReference(ctx, root, part)
			if err != nil {
				return nil, err
			}
			if ref.FieldName != WalletSpecBalanceField {
				err := fmt.Errorf("illegal wallet field in value: %s", ref.FieldName)
				return nil, err
			}
			wallet, _ := root.Wallets.WalletSpec(ref.WalletName)
			valueStrParts[i] = wallet.FieldValue(ref.FieldName).(*big.Int).String()
		}
	}
	valueStr = strings.Join(valueStrParts, " ")

	denomintators := append(append([]string{}, commonDenominations...), additionalDenominators...)
	valueStr = strings.ToLower(valueStr)
	var valueDenomintator string
	for _, den := range denomintators {
		if strings.HasSuffix(valueStr, den) {
			valueStr = strings.TrimSuffix(valueStr, den)
			if !isMathExp(valueStr) {
				err := errors.New("not a math expression in value string")
				return nil, err
			}
			valueDenomintator = den
		}
	}
	if !isMathExp(valueStr) {
		err := errors.New("not a math expression in value string")
		return nil, err
	}
	evaler := NewEvaler()
	var value *big.Int
	if v, err := evaler.Run(valueStr, ExprTypeInterger); err != nil {
		return nil, err
	} else {
		value = v.(*big.Int)
	}
	if len(valueDenomintator) > 0 {
		value = denominateValue(value, valueDenomintator)
	}
	extended := &ExtendedValue{
		Value:       value,
		ValueWei:    ethfw.BigWei(value),
		Denominator: valueDenomintator,
	}
	return extended, nil
}

type ExtendedValue struct {
	Value       *big.Int
	ValueWei    *ethfw.Wei
	Denominator string
}

func denominateValue(val *big.Int, denominator string) *big.Int {
	switch denominator {
	case "wei", "eth":
		return val
	case "gwei":
		v := big.NewInt(1)
		v = v.Mul(val, big.NewInt(1e9))
		v = v.Mul(v, val)
		return v
	case "ether":
		v := big.NewInt(1)
		v = v.Mul(v, big.NewInt(1e9)).Mul(v, big.NewInt(1e9))
		v = v.Mul(v, val)
		return v
	default:
		return val
	}
}

var commonDenominations = []string{
	"wei", "gwei", "ether",
	"eth",
}
