package ethfw

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/AtlantPlatform/ethfw/sol"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TransactFunc func(opts *bind.TransactOpts, contract *common.Address, input []byte) (*types.Transaction, error)

type BoundContract struct {
	*bind.BoundContract

	transactFn TransactFunc
	client     *ethclient.Client
	address    common.Address
	src        *sol.Contract
	abi        abi.ABI
}

func BindContract(client *ethclient.Client, contract *sol.Contract) (*BoundContract, error) {
	if contract == nil {
		err := errors.New("contract must not be nil")
		return nil, err
	}
	parsedABI, err := abi.JSON(bytes.NewReader(contract.ABI))
	if err != nil {
		err = fmt.Errorf("failed to parse contract ABI: %v", err)
		return nil, err
	}
	bound := &BoundContract{
		BoundContract: bind.NewBoundContract(contract.Address, parsedABI, client, client, client),
		client:        client,

		address: contract.Address,
		abi:     parsedABI,
		src:     contract,
	}
	return bound, nil
}

func (contract *BoundContract) SetTransact(fn TransactFunc) {
	contract.transactFn = fn
}

func (contract *BoundContract) SetClient(client *ethclient.Client) {
	contract.client = client
	contract.BoundContract = bind.NewBoundContract(
		contract.address, contract.abi, client, client, client)
}

func (contract *BoundContract) Client() *ethclient.Client {
	return contract.client
}

func (contract *BoundContract) Address() common.Address {
	return contract.address
}

func (contract *BoundContract) SetAddress(address common.Address) {
	contract.address = address
	contract.BoundContract = bind.NewBoundContract(
		address, contract.abi, contract.client, contract.client, contract.client)
}

func (contract *BoundContract) Source() *sol.Contract {
	return contract.src
}

func (contract *BoundContract) ABI() abi.ABI {
	return contract.abi
}

// DeployContract deploys a contract onto the Ethereum blockchain and binds the
// deployment address with a Go wrapper.
func (c *BoundContract) DeployContract(opts *bind.TransactOpts,
	params ...interface{}) (common.Address, *types.Transaction, error) {

	if c.transactFn == nil {
		addr, tx, bound, err := bind.DeployContract(opts, c.abi, common.FromHex(c.src.Bin), c.client, params...)
		if err != nil {
			return addr, tx, err
		}
		c.BoundContract = bound
		return addr, tx, nil
	}

	c.BoundContract = bind.NewBoundContract(common.Address{}, c.abi, c.client, c.client, c.client)
	input, err := c.abi.Pack("", params...)
	if err != nil {
		return common.Address{}, nil, err
	}
	tx, err := c.transactFn(opts, nil, append(common.FromHex(c.src.Bin), input...))
	if err != nil {
		return common.Address{}, nil, err
	}
	c.address = crypto.CreateAddress(opts.From, tx.Nonce())
	c.BoundContract = bind.NewBoundContract(c.address, c.abi, c.client, c.client, c.client)
	return c.address, tx, nil
}

// Transact invokes the (paid) contract method with params as input values.
func (c *BoundContract) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {

	if c.transactFn == nil {
		return c.BoundContract.Transact(opts, method, params...)
	}

	input, err := c.abi.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	return c.transactFn(opts, &c.address, input)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (c *BoundContract) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	if c.transactFn == nil {
		return c.BoundContract.Transfer(opts)
	}
	return c.transactFn(opts, &c.address, nil)
}
