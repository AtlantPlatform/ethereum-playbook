// Copyright 2017, 2018 Tensigma Ltd. All rights reserved.
// Use of this source code is governed by Microsoft Reference Source
// License (MS-RSL) that can be found in the LICENSE file.

package sol

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhichSolc(t *testing.T) {
	assert := assert.New(t)
	path, err := WhichSolc()
	assert.NoError(err)
	if !assert.NotEmpty(path) {
		t.FailNow()
	}
}

func TestCompile(t *testing.T) {
	assert := assert.New(t)
	prepare(`contract mortal {
			    address owner;
			    function mortal() { owner = msg.sender; }
			    function kill() { if (msg.sender == owner) suicide(owner); }
			}

			contract greeter is mortal {
			    string greeting;
			    function greeter(string _greeting) public {
			        greeting = _greeting;
			    }
			    function greet() constant returns (string) {
			        return greeting;
			    }
			}`)
	defer cleanup()
	solcPath, err := WhichSolc()
	orPanic(err)
	c, err := NewSolCompiler(solcPath)
	orPanic(err)
	contracts, err := c.Compile("", "test.sol")
	if !assert.NoError(err) {
		return
	}

	if !assert.Contains(contracts, "mortal") {
		return
	}
	assert.NotEmpty(contracts["mortal"].CompilerVersion)
	assert.NotEmpty(contracts["mortal"].ABI)
	assert.NotEmpty(contracts["mortal"].Bin)
	assert.Equal(contracts["mortal"].Name, "mortal")
	assert.Equal(contracts["mortal"].SourcePath, "test.sol")

	if !assert.Contains(contracts, "greeter") {
		return
	}
	assert.NotEmpty(contracts["greeter"].CompilerVersion)
	assert.NotEmpty(contracts["greeter"].ABI)
	assert.NotEmpty(contracts["greeter"].Bin)
	assert.Equal(contracts["greeter"].Name, "greeter")
	assert.Equal(contracts["greeter"].SourcePath, "test.sol")
}

func cleanup() {
	os.Remove("test.sol")
}

func prepare(sol string) {
	err := ioutil.WriteFile("test.sol", []byte(sol), 0644)
	orPanic(err)
}

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}
