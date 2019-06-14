// Copyright 2017-2019 Tensigma Ltd. All rights reserved.
// Use of this source code is governed by Microsoft Reference Source
// License (MS-RSL) that can be found in the LICENSE file.

package ethfw

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/AtlantPlatform/ethfw/sol"
)

func ContractDeployBin(c *sol.Contract, params ...interface{}) ([]byte, error) {
	parsedABI, err := abi.JSON(bytes.NewReader(c.ABI))
	if err != nil {
		err = fmt.Errorf("failed to parse contract ABI: %v", err)
		return nil, err
	}
	input, err := parsedABI.Pack("", params...)
	if err != nil {
		err = fmt.Errorf("failed to pack contract params: %v", err)
		return nil, err
	}
	bin := append([]byte(c.Bin), input...)
	return bin, nil
}

func ContractCallBin(c *sol.Contract, params ...interface{}) (abi.ABI, []byte, error) {
	parsedABI, err := abi.JSON(bytes.NewReader(c.ABI))
	if err != nil {
		err = fmt.Errorf("failed to parse contract ABI: %v", err)
		return abi.ABI{}, nil, err
	}
	input, err := parsedABI.Pack("", params...)
	if err != nil {
		err = fmt.Errorf("failed to pack contract params: %v", err)
		return abi.ABI{}, nil, err
	}
	return parsedABI, input, nil
}

func ContractAddress(sender common.Address, nonce uint64) common.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{sender, nonce})
	return common.BytesToAddress(crypto.Keccak256(data)[12:])
}
