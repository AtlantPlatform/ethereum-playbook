package executor

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/AtlantPlatform/ethereum-playbook/model"
	"github.com/AtlantPlatform/ethfw"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type Executor struct {
	root      *model.Spec
	nodeGroup string

	ethRPC   *rpc.Client
	ethCli   *ethclient.Client
	keycache ethfw.KeyCache
}

func New(ctx model.AppContext, root *model.Spec) (*Executor, error) {
	nodeGroup := ctx.NodeGroup()
	ethRPC, ok := root.Inventory.GetClient(nodeGroup)
	if !ok {
		err := errors.New("no valid RPC client found in the inventory")
		return nil, err
	}
	executor := &Executor{
		root:      root,
		nodeGroup: nodeGroup,
		ethRPC:    ethRPC,
		ethCli:    ethclient.NewClient(ethRPC),
		keycache:  ctx.KeyCache(),
	}
	return executor, nil
}

func (e *Executor) RunCommand(ctx model.AppContext, cmdName string) ([]*CommandResult, bool) {
	if cmdSpec, ok := e.root.CallCmds[cmdName]; ok {
		return e.runCallCmd(ctx, cmdSpec), true
	}
	if cmdSpec, ok := e.root.ReadCmds[cmdName]; ok {
		return e.runReadCmd(ctx, cmdSpec), true
	}
	if cmdSpec, ok := e.root.WriteCmds[cmdName]; ok {
		return e.runWriteCmd(ctx, cmdSpec), true
	}
	return nil, false
}

type CommandResult struct {
	Wallet string
	Result interface{}
	Error  error
}

func (e *Executor) runCallCmd(ctx context.Context, cmdSpec *model.CallCmdSpec) []*CommandResult {
	matchingWallets := cmdSpec.MatchingWallets()
	results := make([]*CommandResult, len(matchingWallets))
	if len(matchingWallets) > 0 {
		for offset, walletSpec := range matchingWallets {
			walletAddress := common.HexToAddress(walletSpec.Address)
			params := replaceWalletPlaceholders(cmdSpec.ParamValues(), walletAddress)
			params = replaceWalletFieldReferences(params, e.root)
			result := &CommandResult{
				Wallet: walletSpec.Address,
			}
			result.Error = e.ethRPC.CallContext(ctx, &result.Result, cmdSpec.Method, params...)
			results[offset] = result
		}
		return results
	}
	result := &CommandResult{}
	params := replaceWalletFieldReferences(cmdSpec.ParamValues(), e.root)
	result.Error = e.ethRPC.CallContext(ctx, &result.Result, cmdSpec.Method, params...)
	results = append(results, result)
	return results
}

func (e *Executor) runReadCmd(ctx context.Context, cmdSpec *model.ReadCmdSpec) []*CommandResult {
	if !cmdSpec.Instance.IsDeployed() {
		return []*CommandResult{{
			Error: errors.New("contract instance is not deployed yet"),
		}}
	}
	binding := cmdSpec.Instance.BoundContract()
	binding.SetClient(e.ethCli)
	matchingWallets := cmdSpec.MatchingWallets()
	results := make([]*CommandResult, len(matchingWallets))
	if len(matchingWallets) > 0 {
		for offset, walletSpec := range matchingWallets {
			walletAddress := common.HexToAddress(walletSpec.Address)
			params := replaceWalletPlaceholders(cmdSpec.ParamValues(), walletAddress)
			params = replaceWalletFieldReferences(params, e.root)
			result := &CommandResult{
				Wallet: walletSpec.Address,
			}
			opts := &bind.CallOpts{
				From:    walletAddress,
				Context: ctx,
			}
			result.Error = binding.Call(opts, &result.Result, cmdSpec.Method, params...)
			results[offset] = result
		}
		return results
	}
	opts := &bind.CallOpts{
		Context: ctx,
	}
	result := &CommandResult{}
	params := replaceWalletFieldReferences(cmdSpec.ParamValues(), e.root)
	result.Error = binding.Call(opts, &result.Result, cmdSpec.Method, params...)
	results = append(results, result)
	return results
}

func (e *Executor) runWriteCmd(ctx model.AppContext, cmdSpec *model.WriteCmdSpec) []*CommandResult {
	var binding *ethfw.BoundContract
	var denominations []string
	if cmdSpec.Instance != nil {
		// TODO: must do this for all contract instances
		binding = cmdSpec.Instance.BoundContract()
		binding.SetClient(e.ethCli)
		if cmdSpec.Instance.IsDeployed() {
			if name := cmdSpec.Instance.FetchSymbolName(ctx); len(name) > 0 {
				denominations = append(denominations, strings.ToLower(name))
			}
		}
	}
	result := &CommandResult{}
	wallet := cmdSpec.MatchingWallet()
	account := common.HexToAddress(wallet.Address)
	balance, err := e.ethCli.BalanceAt(ctx, account, nil)
	if err != nil {
		result.Error = err
		return []*CommandResult{result}
	}
	wallet.Balance = balance
	nonce, err := e.ethCli.PendingNonceAt(ctx, account)
	if err != nil {
		result.Error = err
		return []*CommandResult{result}
	}
	if len(cmdSpec.To) > 0 {
		to := common.HexToAddress(cmdSpec.To)
		value, err := cmdSpec.Value.Parse(ctx, e.root, denominations)
		if err != nil {
			result.Error = err
			return []*CommandResult{result}
		}
		// log.Println("sending", value.Value.String(), gasLimit, gasPrice.ToInt().String())
		tx := types.NewTransaction(nonce, to, value.Value, gasLimit, gasPrice.ToInt(), nil)
		pk, ok := e.keycache.PrivateKey(account, wallet.Password)
		if !ok {
			result.Error = errors.New("failed to get account private key")
			return []*CommandResult{result}
		}
		signer := types.NewEIP155Signer(chainID)
		signedTx, err := types.SignTx(tx, signer, pk)
		if err != nil {
			result.Error = err
			return []*CommandResult{result}
		}
		result.Error = e.ethCli.SendTransaction(ctx, signedTx)
		result.Result = "tx:" + strings.ToLower(tx.Hash().Hex())
		return []*CommandResult{result}
	}
	return nil
}

const gasLimit = 1000000

var chainID = big.NewInt(1)

var gasPrice = ethfw.Gwei(40)

func replaceWalletPlaceholders(params []interface{}, walletAddress common.Address) []interface{} {
	newParams := append([]interface{}{}, params...)
	for i, param := range newParams {
		if addr, ok := param.(common.Address); ok {
			if bytes.Equal(addr.Bytes(), model.PlaceholderAddr.Bytes()) {
				newParams[i] = walletAddress
			}
		}
	}
	return newParams
}

func replaceWalletFieldReferences(params []interface{}, root *model.Spec) []interface{} {
	newParams := append([]interface{}{}, params...)
	for i, param := range newParams {
		if ref, ok := param.(*model.WalletFieldReference); ok {
			wallet, _ := root.Wallets.WalletSpec(ref.WalletName)
			newParams[i] = wallet.FieldValue(ref.FieldName)
		}
	}
	return newParams
}
