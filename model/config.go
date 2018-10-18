package model

import (
	"math/big"
	"strconv"

	"github.com/AtlantPlatform/ethfw"
	log "github.com/Sirupsen/logrus"
)

type ConfigSpec struct {
	GasPrice string `yaml:"gasPrice"`
	GasLimit string `yaml:"gasLimit"`
	ChainID  string `yaml:"chainID"`

	SpecDir string `yaml:"-"`
}

var DefaultConfigSpec = &ConfigSpec{
	// mainnet: 1
	// others: https://eips.ethereum.org/EIPS/eip-155
	ChainID:  "1",
	GasPrice: ethfw.Gwei(40).String(),
	// hard limit, real limit is estimated
	GasLimit: "10000000",
}

func (spec *ConfigSpec) Validate() bool {
	validateLog := log.WithFields(log.Fields{
		"section": "ConfigSpec",
	})
	if len(spec.GasPrice) > 0 {
		if _, ok := spec.GasPriceInt(); !ok {
			validateLog.Errorln("failed to parse gas_price")
		}
	} else {
		spec.GasPrice = DefaultConfigSpec.GasPrice
	}
	if len(spec.GasLimit) > 0 {
		if _, err := spec.GasLimitInt(); err != nil {
			validateLog.WithError(err).Errorln("failed to parse gas_limit")
		}
	} else {
		spec.GasLimit = DefaultConfigSpec.GasLimit
	}
	if len(spec.ChainID) > 0 {
		if _, ok := spec.ChainIDInt(); !ok {
			validateLog.Errorln("failed to parse chain_id")
		}
	} else {
		spec.ChainID = DefaultConfigSpec.ChainID
	}
	return true
}

func (spec *ConfigSpec) GasLimitInt() (uint64, error) {
	i, err := strconv.ParseUint(spec.GasLimit, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint64(i), nil
}

func (spec *ConfigSpec) GasPriceInt() (*big.Int, bool) {
	return big.NewInt(0).SetString(spec.GasPrice, 10)
}

func (spec *ConfigSpec) ChainIDInt() (*big.Int, bool) {
	return big.NewInt(0).SetString(spec.ChainID, 10)
}
