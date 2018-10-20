package model

import (
	"path/filepath"
	"strings"

	"github.com/AtlantPlatform/ethfw"
	"github.com/AtlantPlatform/ethfw/sol"
	log "github.com/Sirupsen/logrus"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type Contracts map[string]*ContractSpec

func (contracts Contracts) UseSolc() bool {
	for _, c := range contracts {
		if len(c.SolPath) > 0 {
			return true
		}
	}
	return false
}

func (contracts Contracts) Validate(ctx AppContext, spec *Spec) bool {
	for name, contract := range contracts {
		if !contract.Validate(ctx, name) {
			return false
		}
		for _, instance := range contract.Instances {
			if !instance.Validate(ctx, name, contract.src) {
				return false
			}
		}
	}
	return true
}

func (contracts Contracts) ContractSpec(name string) (*ContractSpec, bool) {
	spec, ok := contracts[name]
	return spec, ok
}

func (contracts Contracts) FindByTokenSymbol(symbol string) (*ContractInstanceSpec, bool) {
	symbol = strings.ToUpper(symbol)
	for _, contract := range contracts {
		for _, instance := range contract.Instances {
			if instance.tokenSymbol == symbol {
				return instance, true
			}
		}
	}
	return nil, false
}

type ContractSpec struct {
	Name      string                  `yaml:"name"`
	SolPath   string                  `yaml:"sol"`
	Instances []*ContractInstanceSpec `yaml:"instances"`

	src *sol.Contract `yaml:"-"`
}

func (spec *ContractSpec) Validate(ctx AppContext, name string) bool {
	validateLog := log.WithFields(log.Fields{
		"section":  "Contracts",
		"contract": name,
	})
	if len(spec.Name) == 0 {
		validateLog.Errorln("the root contract name must be specified")
		return false
	}
	if len(spec.SolPath) == 0 {
		validateLog.Errorln("contract spec must have the path to .sol file")
		return false
	}
	if !filepath.IsAbs(spec.SolPath) {
		spec.SolPath = filepath.FromSlash(spec.SolPath)
	}
	if !isFile(filepath.Join(ctx.SpecDir(), spec.SolPath)) {
		validateLog.Errorln("sol file is not found or cannot be read")
		return false
	}
	contracts, err := ctx.SolcCompiler().Compile(ctx.SpecDir(), spec.SolPath)
	if err != nil {
		validateLog.WithError(err).Errorln("sol files compilation failed")
		return false
	}
	src, ok := contracts[spec.Name]
	if !ok {
		validateLog.WithField("name", spec.Name).
			Errorln("specified contract cannot be found in Solidity sources")
		return false
	}
	spec.src = src
	return true
}

type ContractInstanceSpec struct {
	Name    string `yaml:"contract"`
	Address string `yaml:"address"`

	binding     *ethfw.BoundContract `yaml:"-"`
	tokenSymbol string               `yaml:"-"`
}

func (spec *ContractInstanceSpec) Validate(ctx AppContext, name string, src *sol.Contract) bool {
	validateLog := log.WithFields(log.Fields{
		"section":  "ContractInstances",
		"contract": name,
	})
	if len(spec.Address) == 0 || spec.Address == ZeroAddress {
		if len(spec.Name) == 0 {
			validateLog.Errorln("contract instance cannot be deployed without name nor address specified")
			return false
		}
	} else if !common.IsHexAddress(spec.Address) {
		validateLog.Errorln("contract instance address is not valid (must be hex string starting from 0x)")
		return false
	}
	binding, err := ethfw.BindContract(nil, src)
	if err != nil {
		validateLog.WithError(err).Errorln("failed to create contract instance binding")
		return false
	}
	spec.binding = binding
	return true
}

func (spec *ContractInstanceSpec) TokenSymbol() string {
	return spec.tokenSymbol
}

func (spec *ContractInstanceSpec) FetchTokenSymbol(ctx AppContext) string {
	if spec.binding != nil && spec.binding.Client() != nil {
		callOpts := &bind.CallOpts{
			Context: ctx,
		}
		var symbol string
		if err := spec.binding.Call(callOpts, &symbol, "symbol"); err == nil {
			spec.tokenSymbol = strings.ToUpper(symbol)
		}
	}
	return spec.tokenSymbol
}

func (spec *ContractInstanceSpec) BoundContract() *ethfw.BoundContract {
	return spec.binding
}

func (spec *ContractInstanceSpec) IsDeployed() bool {
	return len(spec.Address) > 0 && spec.Address != ZeroAddress
}
