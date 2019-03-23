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
			ref, err := newWalletFieldReference(root, part)
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
		if isArgRef(part) {
			ref, err := newArgReference(ctx, part)
			if err != nil {
				return nil, err
			}
			if ref.ArgID < 0 {
				err := errors.New("insufficient arguments provided")
				return nil, err
			}
			valueStrParts[i] = ctx.AppCommandArgs()[ref.ArgID]
		}
	}
	valueStr = strings.Join(valueStrParts, " ")

	denomintators := append(append([]string{}, commonDenominations...), additionalDenominators...)
	valueStr = strings.ToLower(valueStr)
	var valueDenomintator string
	for _, den := range denomintators {
		if strings.HasSuffix(valueStr, " "+den) {
			valueStr = strings.TrimSuffix(valueStr, " "+den)
			if !isMathExp(valueStr) {
				err := fmt.Errorf("not a math expression in value string: %s", valueStr)
				return nil, err
			}
			valueDenomintator = den
		}
	}
	if !isMathExp(valueStr) {
		err := fmt.Errorf("not a math expression in value string: %s", valueStr)
		return nil, err
	}
	evaler := NewEvaler()
	var value *big.Int

	result, err := evaler.Run(valueStr, ExprTypeInterger)
	if err != nil {
		return nil, err
	}
	value = result.(*big.Int)

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

func (v Valuer) CountArgsUsing(set map[int]struct{}) {
	valueStr := string(v)
	valueStrParts := strings.Split(valueStr, " ")
	for _, part := range valueStrParts {
		if isArgRef(part) {
			if argID, err := argReferenceID(part); err == nil {
				set[argID] = struct{}{}
			}
		}
	}
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

func IsCommonDenominator(name string) bool {
	for _, den := range commonDenominations {
		if den == name {
			return true
		}
	}
	return false
}

var commonDenominations = []string{
	"wei", "gwei", "ether",
	"eth",
}
