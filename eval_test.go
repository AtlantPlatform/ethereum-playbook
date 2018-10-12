package main

import (
	"go/token"
	"log"
	"testing"

	eval "github.com/sbinet/go-eval"
	"github.com/stretchr/testify/assert"
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

func (c *Evaler) Run(expr string) (interface{}, error) {
	code, err := c.world.Compile(c.fset, expr)
	if err != nil {
		return nil, err
	}
	log.Println("Code type:", code.Type())
	result, err := code.Run()
	if err != nil {
		return nil, err
	}
	log.Println("Result:", result.String())
	return result, nil
}

func TestEval(t *testing.T) {
	assert := assert.New(t)
	c := NewEvaler()
	_, err := c.Run("50")
	assert.NoError(err)
	_, err = c.Run("-50")
	assert.NoError(err)
	_, err = c.Run("1")
	assert.NoError(err)
	_, err = c.Run("0xFF")
	assert.NoError(err)
	_, err = c.Run("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	assert.NoError(err)
	_, err = c.Run("50.9999999999999999999999999999999999999999999999991 * 1e99")
	assert.NoError(err)
	_, err = c.Run("true && true")
	assert.NoError(err)
	_, err = c.Run("50 / 10")
	assert.NoError(err)
}
