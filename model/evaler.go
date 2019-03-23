package model

import (
	"errors"
	"fmt"
	"go/token"
	"math/big"

	eval "github.com/sbinet/go-eval"
)

type Evaler struct {
	fset  *token.FileSet
	world *eval.World
}

func NewEvaler() *Evaler {
	return &Evaler{
		fset:  token.NewFileSet(),
		world: eval.NewWorld(),
	}
}

func (c *Evaler) Run(expr string, expected ...ExprType) (interface{}, error) {
	code, err := c.world.Compile(c.fset, expr)
	if err != nil {
		return nil, err
	} else if code.Type() == nil {
		return nil, errors.New("empty value")
	}
	var exprType ExprType
	switch typ := code.Type().String(); typ {
	case "ideal integer":
		exprType = ExprTypeInterger
	case "ideal float":
		exprType = ExprTypeFloat
	case "bool":
		exprType = ExprTypeBool
	default:
		return nil, fmt.Errorf("unsupported expression type: %s", typ)
	}
	// this will be called before return, because initially the type may look like float,
	// but be representable as a integer.
	checkExpected := func(typ ExprType) error {
		if len(expected) > 0 {
			for _, e := range expected {
				if e == typ {
					return nil
				}
			}
			return fmt.Errorf("unexpected expression type: %s", typ)
		}
		return nil
	}
	result, err := code.Run()
	if err != nil {
		return nil, err
	}
	exprValue := result.String()
	switch exprType {
	case ExprTypeBool:
		if err := checkExpected(exprType); err != nil {
			return nil, err
		}
		return bool(exprValue == "true"), nil
	case ExprTypeFloat:
		f, _, err := big.ParseFloat(exprValue, 10, big.MaxPrec, big.ToZero)
		if err != nil {
			panic("failed to parse float string: " + err.Error())
		}
		if f.IsInt() {
			exprType = ExprTypeInterger
			if err := checkExpected(exprType); err != nil {
				return nil, err
			}
			b, _ := f.Int(nil)
			return b, nil
		}
		if err := checkExpected(exprType); err != nil {
			return nil, err
		}
		return f, nil
	case ExprTypeInterger:
		b, ok := big.NewInt(0).SetString(exprValue, 10)
		if !ok {
			panic("failed to parse integer string")
		}
		if err := checkExpected(exprType); err != nil {
			return nil, err
		}
		return b, nil
	default:
		err := fmt.Errorf("unsupported value type: %s", exprType)
		return nil, err
	}
}

type ExprType string

const (
	ExprTypeBool     ExprType = "bool"
	ExprTypeFloat    ExprType = "float"
	ExprTypeInterger ExprType = "integer"
)
